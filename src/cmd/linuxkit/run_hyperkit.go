package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
	"github.com/moby/hyperkit/go"
	"github.com/satori/go.uuid"
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
	diskSzFlag := flags.String("disk-size", "", "Size of Disk in MB (or GB if 'G' is appended)")
	disk := flags.String("disk", "", "Path to disk image to use")
	data := flags.String("data", "", "Metadata to pass to VM (either a path to a file or a string)")
	ipStr := flags.String("ip", "", "IP address for the VM")
	state := flags.String("state", "", "Path to directory to keep VM state in")
	vsockports := flags.String("vsock-ports", "", "List of vsock ports to forward from the guest on startup (comma separated). A unix domain socket for each port will be created in the state directory")
	startVPNKit := flags.Bool("start-vpnkit", false, "Launch a new VPNKit instead of reusing the instance from Docker for Mac. The new instance will be on a separate internal network. This enables IP port forwarding from the host to the guest if the guest supports it.")
	vpnKitDefaultSocket := filepath.Join(os.Getenv("HOME"), "Library/Containers/com.docker.docker/Data/s50")
	vpnKitEthernetSocket := flags.String("vpnkit-socket", vpnKitDefaultSocket, "Path to VPNKit ethernet socket. The Docker for Mac socket is used by default. Overridden if -start-vpnkit is set.")

	if err := flags.Parse(args); err != nil {
		log.Fatal("Unable to parse args")
	}
	remArgs := flags.Args()
	if len(remArgs) == 0 {
		fmt.Println("Please specify the prefix to the image to boot\n")
		flags.Usage()
		os.Exit(1)
	}
	prefix := remArgs[0]

	if *state == "" {
		*state = prefix + "-state"
	}
	if err := os.MkdirAll(*state, 0755); err != nil {
		log.Fatalf("Could not create state directory: %v", err)
	}

	diskSz, err := getDiskSizeMB(*diskSzFlag)
	if err != nil {
		log.Fatalf("Could parse disk-size %s: %v", *diskSzFlag, err)
	}

	isoPath := ""
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
		isoPath = filepath.Join(*state, "data.iso")
		if err := WriteMetadataISO(isoPath, d); err != nil {
			log.Fatalf("Cannot write user data ISO: %v", err)
		}
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
	cmdline, err := ioutil.ReadFile(prefix + "-cmdline")
	if err != nil {
		log.Fatalf("Cannot open cmdline file: %v", err)
	}

	if diskSz != 0 && *disk == "" {
		*disk = filepath.Join(*state, "disk.img")
	}

	var vpnKitPortSocket string
	var vpnKitProcess *os.Process

	// Launch new VPNKit if needed
	if *startVPNKit {
		*vpnKitEthernetSocket = filepath.Join(*state, "vpnkit_eth.sock")
		vpnKitPortSocket = filepath.Join(*state, "vpnkit_port.sock")
		vsockSocket := filepath.Join(*state, "connect")
		vpnKitProcess, err = launchVPNKit(*vpnKitEthernetSocket, vsockSocket, vpnKitPortSocket)
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
	}

	h, err := hyperkit.New(*hyperkitPath, *vpnKitEthernetSocket, *state)
	if err != nil {
		log.Fatalln("Error creating hyperkit: ", err)
	}

	if h.VSockPorts, err = stringToIntArray(*vsockports, ","); err != nil {
		log.Fatalln("Unable to parse vsock-ports: ", err)
	}

	h.Kernel = prefix + "-kernel"
	h.Initrd = prefix + "-initrd.img"
	h.VPNKitKey = vpnKitKey
	h.UUID = vmUUID
	h.DiskImage = *disk
	h.ISOImage = isoPath
	h.VSock = true
	h.CPUs = *cpus
	h.Memory = *mem
	h.DiskSize = diskSz

	// Add 9p and vsock for port forwarding if VPNKit is launched automatically
	if *startVPNKit {
		// The guest will use this 9P mount to configure which ports to forward
		h.Sockets9P = []hyperkit.Socket9P{{Path: vpnKitPortSocket, Tag: "port"}}
		// VSOCK port 62373 is used to pass traffic from host->guest
		h.VSockPorts = append(h.VSockPorts, 62373)
	}

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
