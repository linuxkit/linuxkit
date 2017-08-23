package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/moby/hyperkit/go"
	"github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

const (
	hyperkitNetworkingNone         string = "none"
	hyperkitNetworkingDockerForMac        = "docker-for-mac"
	hyperkitNetworkingVPNKit              = "vpnkit"
	hyperkitNetworkingVMNet               = "vmnet"
	hyperkitNetworkingDefault             = hyperkitNetworkingDockerForMac
)

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
	data := flags.String("data", "", "Metadata to pass to VM (either a path to a file or a string)")
	ipStr := flags.String("ip", "", "IP address for the VM")
	state := flags.String("state", "", "Path to directory to keep VM state in")
	vsockports := flags.String("vsock-ports", "", "List of vsock ports to forward from the guest on startup (comma separated). A unix domain socket for each port will be created in the state directory")
	networking := flags.String("networking", hyperkitNetworkingDefault, "Networking mode. Valid options are 'default', 'docker-for-mac', 'vpnkit[,socket-path]', 'vmnet' and 'none'. 'docker-for-mac' connects to the network used by Docker for Mac. 'vpnkit' connects to the VPNKit socket specified. If socket-path is omitted a new VPNKit instance will be started and 'vpnkit_eth.sock' will be created in the state directory. 'vmnet' uses the Apple vmnet framework, requires root/sudo. 'none' disables networking.`")

	// Boot type; we try to determine automatically
	uefiBoot := flags.Bool("uefi", false, "Use UEFI boot")
	isoBoot := flags.Bool("iso", false, "Boot image is an ISO")
	kernelBoot := flags.Bool("kernel", false, "Boot image is kernel+initrd+cmdline 'path'-kernel/-initrd/-cmdline")

	// Paths and settings for UEFI firware
	// Note, the default uses the firmware shipped with Docker for Mac
	fw := flags.String("fw", "/Applications/Docker.app/Contents/Resources/uefi/UEFI.fd", "Path to OVMF firmware for UEFI boot")

	if err := flags.Parse(args); err != nil {
		log.Fatal("Unable to parse args")
	}
	remArgs := flags.Args()
	if len(remArgs) == 0 {
		fmt.Println("Please specify the prefix to the image to boot\n")
		flags.Usage()
		os.Exit(1)
	}
	path := remArgs[0]
	prefix := path

	info, err := os.Stat(path)
	stat := err == nil

	// ignore a directory
	if stat && info.Mode().IsDir() {
		stat = false
	}

	_, err = os.Stat(path + "-kernel")
	statKernel := err == nil

	var isoPaths []string

	// try to autodetect boot type if not specified
	// if the path does not exist, and the kernel does, must be trying to do a kernel boot
	// if the path does exist and ends in ISO, must be trying ISO boot
	if !stat && statKernel && !*isoBoot {
		*kernelBoot = true
	} else if stat && strings.HasSuffix(path, ".iso") && !*kernelBoot {
		*isoBoot = true
	}

	switch {
	case *kernelBoot && *isoBoot:
		log.Fatalf("Cannot specify both kernel and ISO boot together")
	case *kernelBoot:
		if !statKernel {
			log.Fatalf("Cannot find kernel file (%s): %v", path+"-kernel", err)
		}
		_, err = os.Stat(path + "-initrd.img")
		statInitrd := err == nil
		if !statInitrd {
			log.Fatalf("Cannot find initrd file (%s): %v", path+"-initrd.img", err)
		}
	case *isoBoot:
		if !stat {
			log.Fatalf("Cannot find ISO to boot")
		}
		prefix = strings.TrimSuffix(path, ".iso")
		// hyperkit only supports UEFI ISO boot at present
		if !*uefiBoot {
			log.Fatalf("Hyperkit requires --uefi to be set to boot an ISO")
		}
		isoPaths = append(isoPaths, path)
	default:
		if !stat {
			log.Fatalf("Cannot find file %s to boot", path)
		}
		log.Fatalf("Unrecognised boot type, please specify on command line")
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

	if *data != "" {
		var d []byte
		if _, err := os.Stat(*data); os.IsNotExist(err) {
			d = []byte(*data)
		} else {
			d, err = ioutil.ReadFile(*data)
			if err != nil {
				log.Fatalf("Cannot read user data: %v", err)
			}
		}
		isoPath := filepath.Join(*state, "data.iso")
		if err := WriteMetadataISO(isoPath, d); err != nil {
			log.Fatalf("Cannot write user data ISO: %v", err)
		}
		isoPaths = append(isoPaths, isoPath)
	}

	vpnKitKey := ""
	if *ipStr != "" {
		// If an IP address was requested construct a "special" UUID
		// for the VM.
		if ip := net.ParseIP(*ipStr); len(ip) > 0 {
			uuid := make([]byte, 16)
			uuid[12] = ip.To4()[0]
			uuid[13] = ip.To4()[1]
			uuid[14] = ip.To4()[2]
			uuid[15] = ip.To4()[3]
			vpnKitKey = fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:])
		}
	}

	// Generate new UUID, otherwise /sys/class/dmi/id/product_uuid is identical on all VMs
	vmUUID := uuid.NewV4().String()

	// Run
	var cmdline []byte
	if *kernelBoot {
		cmdline, err = ioutil.ReadFile(prefix + "-cmdline")
		if err != nil {
			log.Fatalf("Cannot open cmdline file: %v", err)
		}
	}

	// Create new HyperKit instance (w/o networking for now)
	h, err := hyperkit.New(*hyperkitPath, "", *state)
	if err != nil {
		log.Fatalln("Error creating hyperkit: ", err)
	}

	for i, d := range disks {
		id := ""
		if i != 0 {
			id = strconv.Itoa(i)
		}
		if d.Size != 0 && d.Path == "" {
			d.Path = filepath.Join(*state, "disk"+id+".img")
		}
		if d.Path == "" {
			log.Fatalf("disk specified with no size or name")
		}
		hd := hyperkit.DiskConfig{Path: d.Path, Size: d.Size}
		h.Disks = append(h.Disks, hd)
	}

	if h.VSockPorts, err = stringToIntArray(*vsockports, ","); err != nil {
		log.Fatalln("Unable to parse vsock-ports: ", err)
	}

	// Select network mode
	var vpnKitProcess *os.Process
	if *networking == "" || *networking == "default" {
		dflt := hyperkitNetworkingDefault
		networking = &dflt
	}
	netMode := strings.SplitN(*networking, ",", 2)

	switch netMode[0] {
	case hyperkitNetworkingDockerForMac:
		h.VPNKitSock = filepath.Join(os.Getenv("HOME"), "Library/Containers/com.docker.docker/Data/s50")
	case hyperkitNetworkingVPNKit:
		if len(netMode) > 1 {
			// Socket path specified, try to use existing VPNKit instance
			h.VPNKitSock = netMode[1]
		} else {
			// Start new VPNKit instance
			h.VPNKitSock = filepath.Join(*state, "vpnkit_eth.sock")
			vpnKitPortSocket := filepath.Join(*state, "vpnkit_port.sock")
			vsockSocket := filepath.Join(*state, "connect")
			vpnKitProcess, err = launchVPNKit(h.VPNKitSock, vsockSocket, vpnKitPortSocket)
			if err != nil {
				log.Fatalln("Unable to start vpnkit: ", err)
			}
			defer func() {
				if vpnKitProcess != nil {
					err := vpnKitProcess.Kill()
					if err != nil {
						log.Println(err)
					}
				}
			}()
			// The guest will use this 9P mount to configure which ports to forward
			h.Sockets9P = []hyperkit.Socket9P{{Path: vpnKitPortSocket, Tag: "port"}}
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

	if *kernelBoot {
		h.Kernel = prefix + "-kernel"
		h.Initrd = prefix + "-initrd.img"
	} else {
		h.Bootrom = *fw
	}
	h.VPNKitKey = vpnKitKey
	h.UUID = vmUUID
	h.ISOImages = isoPaths
	h.VSock = true
	h.CPUs = *cpus
	h.Memory = *mem

	err = h.Run(string(cmdline))
	if err != nil {
		log.Fatalf("Cannot run hyperkit: %v", err)
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
func launchVPNKit(etherSock string, vsockSock string, portSock string) (*os.Process, error) {
	var err error

	vpnKitPath, err := exec.LookPath("vpnkit")
	if err != nil {
		return nil, fmt.Errorf("Unable to find vpnkit binary")
	}

	etherFile, err := createListenSocket(etherSock)
	if err != nil {
		return nil, err
	}

	portFile, err := createListenSocket(portSock)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(vpnKitPath,
		"--ethernet", "fd:3",
		"--vsock-path", vsockSock,
		"--port", "fd:4")

	cmd.ExtraFiles = append(cmd.ExtraFiles, etherFile)
	cmd.ExtraFiles = append(cmd.ExtraFiles, portFile)

	cmd.Env = os.Environ() // pass env for DEBUG

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	go cmd.Wait() // run in background

	return cmd.Process, nil
}
