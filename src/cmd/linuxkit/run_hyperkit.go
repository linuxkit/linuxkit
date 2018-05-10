package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/moby/hyperkit/go"
	"github.com/moby/vpnkit/go/pkg/vpnkit"
	log "github.com/sirupsen/logrus"
)

const (
	hyperkitNetworkingNone         string = "none"
	hyperkitNetworkingDockerForMac        = "docker-for-mac"
	hyperkitNetworkingVPNKit              = "vpnkit"
	hyperkitNetworkingVMNet               = "vmnet"
	hyperkitNetworkingDefault             = hyperkitNetworkingDockerForMac
)

func init() {
	hyperkit.SetLogger(log.StandardLogger())
}

// Process the run arguments and execute run
func runHyperKit(args []string) {
	flags := flag.NewFlagSet("hyperkit", flag.ExitOnError)
	invoked := filepath.Base(os.Args[0])
	flags.Usage = func() {
		fmt.Printf("USAGE: %s run hyperkit [options] prefix\n\n", invoked)
		fmt.Printf("'prefix' specifies the path to the VM image.\n")
		fmt.Printf("\n")
		fmt.Printf("Options:\n")
		flags.PrintDefaults()
	}
	hyperkitPath := flags.String("hyperkit", "", "Path to hyperkit binary (if not in default location)")
	cpus := flags.Int("cpus", 1, "Number of CPUs")
	mem := flags.Int("mem", 1024, "Amount of memory in MB")
	var disks Disks
	flags.Var(&disks, "disk", "Disk config. [file=]path[,size=1G]")
	data := flags.String("data", "", "String of metadata to pass to VM; error to specify both -data and -data-file")
	dataPath := flags.String("data-file", "", "Path to file containing metadata to pass to VM; error to specify both -data and -data-file")

	if *data != "" && *dataPath != "" {
		log.Fatal("Cannot specify both -data and -data-file")
	}

	ipStr := flags.String("ip", "", "Preferred IPv4 address for the VM.")
	state := flags.String("state", "", "Path to directory to keep VM state in")
	vsockports := flags.String("vsock-ports", "", "List of vsock ports to forward from the guest on startup (comma separated). A unix domain socket for each port will be created in the state directory")
	networking := flags.String("networking", hyperkitNetworkingDefault, "Networking mode. Valid options are 'default', 'docker-for-mac', 'vpnkit[,eth-socket-path[,port-socket-path]]', 'vmnet' and 'none'. 'docker-for-mac' connects to the network used by Docker for Mac. 'vpnkit' connects to the VPNKit socket(s) specified. If no socket path is provided a new VPNKit instance will be started and 'vpnkit_eth.sock' and 'vpnkit_port.sock' will be created in the state directory. 'port-socket-path' is only needed if you want to publish ports on localhost using an existing VPNKit instance. 'vmnet' uses the Apple vmnet framework, requires root/sudo. 'none' disables networking.`")

	vpnkitUUID := flags.String("vpnkit-uuid", "", "Optional UUID used to identify the VPNKit connection. Overrides 'vpnkit.uuid' in the state directory.")
	vpnkitPath := flags.String("vpnkit", "", "Path to vpnkit binary")
	publishFlags := multipleFlag{}
	flags.Var(&publishFlags, "publish", "Publish a vm's port(s) to the host (default [])")

	// Boot type; we try to determine automatically
	uefiBoot := flags.Bool("uefi", false, "Use UEFI boot")
	isoBoot := flags.Bool("iso", false, "Boot image is an ISO")
	squashFSBoot := flags.Bool("squashfs", false, "Boot image is a kernel+squashfs+cmdline")
	kernelBoot := flags.Bool("kernel", false, "Boot image is kernel+initrd+cmdline 'path'-kernel/-initrd/-cmdline")

	// Hyperkit settings
	consoleToFile := flags.Bool("console-file", false, "Output the console to a tty file")

	// Paths and settings for UEFI firmware
	// Note, the default uses the firmware shipped with Docker for Mac
	fw := flags.String("fw", "/Applications/Docker.app/Contents/Resources/uefi/UEFI.fd", "Path to OVMF firmware for UEFI boot")

	if err := flags.Parse(args); err != nil {
		log.Fatal("Unable to parse args")
	}
	remArgs := flags.Args()
	if len(remArgs) == 0 {
		fmt.Println("Please specify the prefix to the image to boot")
		flags.Usage()
		os.Exit(1)
	}
	path := remArgs[0]
	prefix := path

	_, err := os.Stat(path + "-kernel")
	statKernel := err == nil

	var isoPaths []string

	switch {
	case *squashFSBoot:
		if *kernelBoot || *isoBoot {
			log.Fatalf("Please specify only one boot method")
		}
		if !statKernel {
			log.Fatalf("Booting a SquashFS root filesystem requires a kernel at %s", path+"-kernel")
		}
		_, err = os.Stat(path + "-squashfs.img")
		statSquashFS := err == nil
		if !statSquashFS {
			log.Fatalf("Cannot find SquashFS image (%s): %v", path+"-squashfs.img", err)
		}
	case *isoBoot:
		if *kernelBoot {
			log.Fatalf("Please specify only one boot method")
		}
		if !*uefiBoot {
			log.Fatalf("Hyperkit requires --uefi to be set to boot an ISO")
		}
		// We used to auto-detect ISO boot. For backwards compat, append .iso if not present
		isoPath := path
		if !strings.HasSuffix(isoPath, ".iso") {
			isoPath += ".iso"
		}
		_, err = os.Stat(isoPath)
		statISO := err == nil
		if !statISO {
			log.Fatalf("Cannot find ISO image (%s): %v", isoPath, err)
		}
		prefix = strings.TrimSuffix(path, ".iso")
		isoPaths = append(isoPaths, isoPath)
	default:
		// Default to kernel+initrd
		if !statKernel {
			log.Fatalf("Cannot find kernel file: %s", path+"-kernel")
		}
		_, err = os.Stat(path + "-initrd.img")
		statInitrd := err == nil
		if !statInitrd {
			log.Fatalf("Cannot find initrd file (%s): %v", path+"-initrd.img", err)
		}
		*kernelBoot = true
	}

	if *uefiBoot {
		_, err := os.Stat(*fw)
		if err != nil {
			log.Fatalf("Cannot open UEFI firmware file (%s): %v", *fw, err)
		}
	}

	if *state == "" {
		*state = prefix + "-state"
	}
	if err := os.MkdirAll(*state, 0755); err != nil {
		log.Fatalf("Could not create state directory: %v", err)
	}

	metadataPaths, err := CreateMetadataISO(*state, *data, *dataPath)
	if err != nil {
		log.Fatalf("%v", err)
	}
	isoPaths = append(isoPaths, metadataPaths...)

	// Create UUID for VPNKit or reuse an existing one from state dir. IP addresses are
	// assigned to the UUID, so to get the same IP we have to store the initial UUID. If
	// has specified a VPNKit UUID the file is ignored.
	if *vpnkitUUID == "" {
		vpnkitUUIDFile := filepath.Join(*state, "vpnkit.uuid")
		if _, err := os.Stat(vpnkitUUIDFile); os.IsNotExist(err) {
			*vpnkitUUID = uuid.New().String()
			if err := ioutil.WriteFile(vpnkitUUIDFile, []byte(*vpnkitUUID), 0600); err != nil {
				log.Fatalf("Unable to write to %s: %v", vpnkitUUIDFile, err)
			}
		} else {
			uuidBytes, err := ioutil.ReadFile(vpnkitUUIDFile)
			if err != nil {
				log.Fatalf("Unable to read VPNKit UUID from %s: %v", vpnkitUUIDFile, err)
			}
			if tmp, err := uuid.ParseBytes(uuidBytes); err != nil {
				log.Fatalf("Unable to parse VPNKit UUID from %s: %v", vpnkitUUIDFile, err)
			} else {
				*vpnkitUUID = tmp.String()
			}

		}
	}

	// Generate new UUID, otherwise /sys/class/dmi/id/product_uuid is identical on all VMs
	vmUUID := uuid.New().String()

	// Run
	var cmdline string
	if *kernelBoot || *squashFSBoot {
		cmdlineBytes, err := ioutil.ReadFile(prefix + "-cmdline")
		if err != nil {
			log.Fatalf("Cannot open cmdline file: %v", err)
		}
		cmdline = string(cmdlineBytes)
	}

	// Create new HyperKit instance (w/o networking for now)
	h, err := hyperkit.New(*hyperkitPath, "", *state)
	if err != nil {
		log.Fatalln("Error creating hyperkit: ", err)
	}

	if *consoleToFile {
		h.Console = hyperkit.ConsoleFile
	}

	h.UUID = vmUUID
	h.ISOImages = isoPaths
	h.VSock = true
	h.CPUs = *cpus
	h.Memory = *mem

	switch {
	case *kernelBoot:
		h.Kernel = prefix + "-kernel"
		h.Initrd = prefix + "-initrd.img"
	case *squashFSBoot:
		h.Kernel = prefix + "-kernel"
		// Make sure the SquashFS image is the first disk, raw, and virtio
		var rootDisk hyperkit.RawDisk
		rootDisk.Path = prefix + "-squashfs.img"
		rootDisk.Trim = false // This happens to select 'virtio-blk'
		h.Disks = append(h.Disks, &rootDisk)
		cmdline = cmdline + " root=/dev/vda"
	default:
		h.Bootrom = *fw
	}

	for i, d := range disks {
		id := ""
		if i != 0 {
			id = strconv.Itoa(i)
		}
		if d.Size != 0 && d.Path == "" {
			d.Path = filepath.Join(*state, "disk"+id+".raw")
		}
		if d.Path == "" {
			log.Fatalf("disk specified with no size or name")
		}
		hd, err := hyperkit.NewDisk(d.Path, d.Size)
		if err != nil {
			log.Fatalf("NewDisk failed: %v", err)
		}
		h.Disks = append(h.Disks, hd)
	}

	if h.VSockPorts, err = stringToIntArray(*vsockports, ","); err != nil {
		log.Fatalln("Unable to parse vsock-ports: ", err)
	}

	// Select network mode
	var vpnkitProcess *os.Process
	var vpnkitPortSocket string
	if *networking == "" || *networking == "default" {
		dflt := hyperkitNetworkingDefault
		networking = &dflt
	}
	netMode := strings.SplitN(*networking, ",", 3)
	switch netMode[0] {
	case hyperkitNetworkingDockerForMac:
		oldEthSock := filepath.Join(os.Getenv("HOME"), "Library/Containers/com.docker.docker/Data/s50")
		oldPortSock := filepath.Join(os.Getenv("HOME"), "Library/Containers/com.docker.docker/Data/s51")
		newEthSock := filepath.Join(os.Getenv("HOME"), "Library/Containers/com.docker.docker/Data/vpnkit.eth.sock")
		newPortSock := filepath.Join(os.Getenv("HOME"), "Library/Containers/com.docker.docker/Data/vpnkit.port.sock")
		_, err := os.Stat(oldEthSock)
		if err == nil {
			h.VPNKitSock = oldEthSock
			vpnkitPortSocket = oldPortSock
		} else {
			_, err = os.Stat(newEthSock)
			if err != nil {
				log.Fatalln("Cannot find Docker for Mac network sockets. Install Docker or use a different network mode.")
			}
			h.VPNKitSock = newEthSock
			vpnkitPortSocket = newPortSock
		}
	case hyperkitNetworkingVPNKit:
		if len(netMode) > 1 {
			// Socket path specified, try to use existing VPNKit instance
			h.VPNKitSock = netMode[1]
			if len(netMode) > 2 {
				vpnkitPortSocket = netMode[2]
			}
			// The guest will use this 9P mount to configure which ports to forward
			h.Sockets9P = []hyperkit.Socket9P{{Path: vpnkitPortSocket, Tag: "port"}}
			// VSOCK port 62373 is used to pass traffic from host->guest
			h.VSockPorts = append(h.VSockPorts, 62373)
		} else {
			// Start new VPNKit instance
			h.VPNKitSock = filepath.Join(*state, "vpnkit_eth.sock")
			vpnkitPortSocket = filepath.Join(*state, "vpnkit_port.sock")
			vsockSocket := filepath.Join(*state, "connect")
			vpnkitProcess, err = launchVPNKit(*vpnkitPath, h.VPNKitSock, vsockSocket, vpnkitPortSocket)
			if err != nil {
				log.Fatalln("Unable to start vpnkit: ", err)
			}
			defer shutdownVPNKit(vpnkitProcess)
			log.RegisterExitHandler(func() {
				shutdownVPNKit(vpnkitProcess)
			})
			// The guest will use this 9P mount to configure which ports to forward
			h.Sockets9P = []hyperkit.Socket9P{{Path: vpnkitPortSocket, Tag: "port"}}
			// VSOCK port 62373 is used to pass traffic from host->guest
			h.VSockPorts = append(h.VSockPorts, 62373)
		}
	case hyperkitNetworkingVMNet:
		h.VPNKitSock = ""
		h.VMNet = true
	case hyperkitNetworkingNone:
		h.VPNKitSock = ""
	default:
		log.Fatalf("Invalid networking mode: %s", netMode[0])
	}

	h.VPNKitUUID = *vpnkitUUID
	if *ipStr != "" {
		if ip := net.ParseIP(*ipStr); len(ip) > 0 && ip.To4() != nil {
			h.VPNKitPreferredIPv4 = ip.String()
		} else {
			log.Fatalf("Unable to parse IPv4 address: %v", *ipStr)
		}
	}

	// Publish ports if requested and VPNKit is used
	if len(publishFlags) != 0 {
		switch netMode[0] {
		case hyperkitNetworkingDockerForMac, hyperkitNetworkingVPNKit:
			if vpnkitPortSocket == "" {
				log.Fatalf("The VPNKit Port socket path is required to publish ports")
			}
			f, err := vpnkitPublishPorts(h, publishFlags, vpnkitPortSocket)
			if err != nil {
				log.Fatalf("Publish ports failed with: %v", err)
			}
			defer f()
			log.RegisterExitHandler(f)
		default:
			log.Fatalf("Port publishing requires %q or %q networking mode", hyperkitNetworkingDockerForMac, hyperkitNetworkingVPNKit)
		}
	}

	err = h.Run(cmdline)
	if err != nil {
		log.Fatalf("Cannot run hyperkit: %v", err)
	}
}

func shutdownVPNKit(process *os.Process) {
	if process == nil {
		return
	}

	if err := process.Kill(); err != nil {
		log.Println(err)
	}
}

// createListenSocket creates a new unix domain socket and returns the open file
func createListenSocket(path string) (*os.File, error) {
	os.Remove(path)
	conn, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		return nil, fmt.Errorf("unable to create socket: %v", err)
	}
	f, err := conn.File()
	if err != nil {
		return nil, err
	}
	return f, nil
}

// launchVPNKit starts a new instance of VPNKit. Ethernet socket and port socket
// will be created and passed to VPNKit. The VSOCK socket should be created
// by HyperKit when it starts.
func launchVPNKit(vpnkitPath, etherSock, vsockSock, portSock string) (*os.Process, error) {
	var err error

	if vpnkitPath == "" {
		vpnkitPath, err = exec.LookPath("vpnkit")
		if err != nil {
			return nil, fmt.Errorf("Unable to find vpnkit binary")
		}
	}

	etherFile, err := createListenSocket(etherSock)
	if err != nil {
		return nil, err
	}

	portFile, err := createListenSocket(portSock)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(vpnkitPath,
		"--ethernet", "fd:3",
		"--vsock-path", vsockSock,
		"--port", "fd:4")

	cmd.ExtraFiles = append(cmd.ExtraFiles, etherFile)
	cmd.ExtraFiles = append(cmd.ExtraFiles, portFile)

	cmd.Env = os.Environ() // pass env for DEBUG

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Debugf("Starting vpnkit: %v", cmd.Args)
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	go cmd.Wait() // run in background

	return cmd.Process, nil
}

// vpnkitPublishPorts instructs VPNKit to expose ports from the VM on localhost
// Pre-register the VM with VPNKit using the UUID. This gives the IP
// address (if not specified) allowing us to publish ports. It returns
// a function which should be called to clean up once the VM stops.
func vpnkitPublishPorts(h *hyperkit.HyperKit, publishFlags multipleFlag, portSocket string) (func(), error) {
	ctx := context.Background()

	vpnkitUUID, err := uuid.Parse(h.VPNKitUUID)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse VPNKit UUID %s: %v", h.VPNKitUUID, err)
	}

	localhost := net.ParseIP("127.0.0.1")
	if localhost == nil {
		return nil, fmt.Errorf("Failed to parse 127.0.0.1")
	}

	log.Debugf("Creating new VPNKit VMNet on %s", h.VPNKitSock)
	vmnet, err := vpnkit.NewVmnet(ctx, h.VPNKitSock)
	if err != nil {
		return nil, fmt.Errorf("NewVmnet failed: %v", err)
	}
	defer vmnet.Close()

	// Register with VPNKit
	var vif *vpnkit.Vif
	if h.VPNKitPreferredIPv4 == "" {
		log.Debugf("Creating VPNKit VIF for %v", vpnkitUUID)
		vif, err = vmnet.ConnectVif(vpnkitUUID)
		if err != nil {
			return nil, fmt.Errorf("Connection to Vif failed: %v", err)
		}
	} else {
		ip := net.ParseIP(h.VPNKitPreferredIPv4)
		if ip == nil {
			return nil, fmt.Errorf("Failed to parse IP: %s", h.VPNKitPreferredIPv4)
		}
		log.Debugf("Creating VPNKit VIF for %v ip=%v", vpnkitUUID, ip)
		vif, err = vmnet.ConnectVifIP(vpnkitUUID, ip)
		if err != nil {
			return nil, fmt.Errorf("Connection to Vif with IP failed: %v", err)
		}
	}
	log.Debugf("VPNKit UUID:%s IP: %v", vpnkitUUID, vif.IP)

	log.Debugf("Connecting to VPNKit on %s", portSocket)
	c, err := vpnkit.NewConnection(context.Background(), portSocket)
	if err != nil {
		return nil, fmt.Errorf("Connection to VPNKit failed: %v", err)
	}

	// Publish ports
	var ports []*vpnkit.Port
	for _, publish := range publishFlags {
		p, err := NewPublishedPort(publish)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse port publish %s: %v", publish, err)
		}

		log.Debugf("Publishing %s", publish)
		vp := vpnkit.NewPort(c, p.Protocol, localhost, p.Host, vif.IP, p.Guest)
		if err = vp.Expose(context.Background()); err != nil {
			return nil, fmt.Errorf("Failed to expose port %s: %v", publish, err)
		}
		ports = append(ports, vp)
	}

	// Return cleanup function
	return func() {
		for _, vp := range ports {
			vp.Unexpose(context.Background())
		}
	}, nil
}
