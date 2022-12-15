package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"
	hyperkit "github.com/moby/hyperkit/go"
	"github.com/moby/vpnkit/go/pkg/vmnet"
	"github.com/moby/vpnkit/go/pkg/vpnkit"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
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

func runHyperkitCmd() *cobra.Command {
	var (
		hyperkitPath  string
		data          string
		dataPath      string
		ipStr         string
		state         string
		vsockports    string
		networking    string
		vpnkitUUID    string
		vpnkitPath    string
		uefiBoot      bool
		isoBoot       bool
		squashFSBoot  bool
		kernelBoot    bool
		consoleToFile bool
		fw            string
		publishFlags  multipleFlag
	)
	cmd := &cobra.Command{
		Use:   "hyperkit",
		Short: "launch a VM using hyperkit",
		Long: `Launch a VM using hyperkit.
		'prefix' specifies the path to the VM image.
		`,
		Args:    cobra.ExactArgs(1),
		Example: "linuxkit run hyperkit [options] prefix",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]

			if data != "" && dataPath != "" {
				return errors.New("Cannot specify both -data and -data-file")
			}

			prefix := path

			_, err := os.Stat(path + "-kernel")
			statKernel := err == nil

			var isoPaths []string

			switch {
			case squashFSBoot:
				if kernelBoot || isoBoot {
					return fmt.Errorf("Please specify only one boot method")
				}
				if !statKernel {
					return fmt.Errorf("Booting a SquashFS root filesystem requires a kernel at %s", path+"-kernel")
				}
				_, err = os.Stat(path + "-squashfs.img")
				statSquashFS := err == nil
				if !statSquashFS {
					return fmt.Errorf("Cannot find SquashFS image (%s): %v", path+"-squashfs.img", err)
				}
			case isoBoot:
				if kernelBoot {
					return fmt.Errorf("Please specify only one boot method")
				}
				if !uefiBoot {
					return fmt.Errorf("Hyperkit requires --uefi to be set to boot an ISO")
				}
				// We used to auto-detect ISO boot. For backwards compat, append .iso if not present
				isoPath := path
				if !strings.HasSuffix(isoPath, ".iso") {
					isoPath += ".iso"
				}
				_, err = os.Stat(isoPath)
				statISO := err == nil
				if !statISO {
					return fmt.Errorf("Cannot find ISO image (%s): %v", isoPath, err)
				}
				prefix = strings.TrimSuffix(path, ".iso")
				isoPaths = append(isoPaths, isoPath)
			default:
				// Default to kernel+initrd
				if !statKernel {
					return fmt.Errorf("Cannot find kernel file: %s", path+"-kernel")
				}
				_, err = os.Stat(path + "-initrd.img")
				statInitrd := err == nil
				if !statInitrd {
					return fmt.Errorf("Cannot find initrd file (%s): %v", path+"-initrd.img", err)
				}
				kernelBoot = true
			}

			if uefiBoot {
				_, err := os.Stat(fw)
				if err != nil {
					return fmt.Errorf("Cannot open UEFI firmware file (%s): %v", fw, err)
				}
			}

			if state == "" {
				state = prefix + "-state"
			}
			if err := os.MkdirAll(state, 0755); err != nil {
				return fmt.Errorf("Could not create state directory: %v", err)
			}

			metadataPaths, err := CreateMetadataISO(state, data, dataPath)
			if err != nil {
				return fmt.Errorf("%v", err)
			}
			isoPaths = append(isoPaths, metadataPaths...)

			// Create UUID for VPNKit or reuse an existing one from state dir. IP addresses are
			// assigned to the UUID, so to get the same IP we have to store the initial UUID. If
			// has specified a VPNKit UUID the file is ignored.
			if vpnkitUUID == "" {
				vpnkitUUIDFile := filepath.Join(state, "vpnkit.uuid")
				if _, err := os.Stat(vpnkitUUIDFile); os.IsNotExist(err) {
					vpnkitUUID = uuid.New().String()
					if err := os.WriteFile(vpnkitUUIDFile, []byte(vpnkitUUID), 0600); err != nil {
						return fmt.Errorf("Unable to write to %s: %v", vpnkitUUIDFile, err)
					}
				} else {
					uuidBytes, err := os.ReadFile(vpnkitUUIDFile)
					if err != nil {
						return fmt.Errorf("Unable to read VPNKit UUID from %s: %v", vpnkitUUIDFile, err)
					}
					if tmp, err := uuid.ParseBytes(uuidBytes); err != nil {
						return fmt.Errorf("Unable to parse VPNKit UUID from %s: %v", vpnkitUUIDFile, err)
					} else {
						vpnkitUUID = tmp.String()
					}
				}
			}

			// Generate new UUID, otherwise /sys/class/dmi/id/product_uuid is identical on all VMs
			vmUUID := uuid.New().String()

			// Run
			var cmdline string
			if kernelBoot || squashFSBoot {
				cmdlineBytes, err := os.ReadFile(prefix + "-cmdline")
				if err != nil {
					return fmt.Errorf("Cannot open cmdline file: %v", err)
				}
				cmdline = string(cmdlineBytes)
			}

			// Create new HyperKit instance (w/o networking for now)
			h, err := hyperkit.New(hyperkitPath, "", state)
			if err != nil {
				return fmt.Errorf("Error creating hyperkit: %w", err)
			}

			if consoleToFile {
				h.Console = hyperkit.ConsoleFile
			}

			h.UUID = vmUUID
			h.ISOImages = isoPaths
			h.VSock = true
			h.CPUs = cpus
			h.Memory = mem

			switch {
			case kernelBoot:
				h.Kernel = prefix + "-kernel"
				h.Initrd = prefix + "-initrd.img"
			case squashFSBoot:
				h.Kernel = prefix + "-kernel"
				// Make sure the SquashFS image is the first disk, raw, and virtio
				var rootDisk hyperkit.RawDisk
				rootDisk.Path = prefix + "-squashfs.img"
				rootDisk.Trim = false // This happens to select 'virtio-blk'
				h.Disks = append(h.Disks, &rootDisk)
				cmdline = cmdline + " root=/dev/vda"
			default:
				h.Bootrom = fw
			}

			for i, d := range disks {
				id := ""
				if i != 0 {
					id = strconv.Itoa(i)
				}
				if d.Size != 0 && d.Path == "" {
					d.Path = filepath.Join(state, "disk"+id+".raw")
				}
				if d.Path == "" {
					return fmt.Errorf("disk specified with no size or name")
				}
				hd, err := hyperkit.NewDisk(d.Path, d.Size)
				if err != nil {
					return fmt.Errorf("NewDisk failed: %v", err)
				}
				h.Disks = append(h.Disks, hd)
			}

			if h.VSockPorts, err = stringToIntArray(vsockports, ","); err != nil {
				return fmt.Errorf("Unable to parse vsock-ports: %w", err)
			}

			// Select network mode
			var vpnkitProcess *os.Process
			var vpnkitPortSocket string
			if networking == "" || networking == "default" {
				dflt := hyperkitNetworkingDefault
				networking = dflt
			}
			netMode := strings.SplitN(networking, ",", 3)
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
						return errors.New("Cannot find Docker for Mac network sockets. Install Docker or use a different network mode.")
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
					h.VPNKitSock = filepath.Join(state, "vpnkit_eth.sock")
					vpnkitPortSocket = filepath.Join(state, "vpnkit_port.sock")
					vsockSocket := filepath.Join(state, "connect")
					vpnkitProcess, err = launchVPNKit(vpnkitPath, h.VPNKitSock, vsockSocket, vpnkitPortSocket)
					if err != nil {
						return fmt.Errorf("Unable to start vpnkit: %w", err)
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
				return fmt.Errorf("Invalid networking mode: %s", netMode[0])
			}

			h.VPNKitUUID = vpnkitUUID
			if ipStr != "" {
				if ip := net.ParseIP(ipStr); len(ip) > 0 && ip.To4() != nil {
					h.VPNKitPreferredIPv4 = ip.String()
				} else {
					return fmt.Errorf("Unable to parse IPv4 address: %v", ipStr)
				}
			}

			// Publish ports if requested and VPNKit is used
			if len(publishFlags) != 0 {
				switch netMode[0] {
				case hyperkitNetworkingDockerForMac, hyperkitNetworkingVPNKit:
					if vpnkitPortSocket == "" {
						return fmt.Errorf("The VPNKit Port socket path is required to publish ports")
					}
					f, err := vpnkitPublishPorts(h, publishFlags, vpnkitPortSocket)
					if err != nil {
						return fmt.Errorf("Publish ports failed with: %v", err)
					}
					defer f()
					log.RegisterExitHandler(f)
				default:
					return fmt.Errorf("Port publishing requires %q or %q networking mode", hyperkitNetworkingDockerForMac, hyperkitNetworkingVPNKit)
				}
			}

			err = h.Run(cmdline)
			if err != nil {
				return fmt.Errorf("Cannot run hyperkit: %v", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&hyperkitPath, "hyperkit", "", "Path to hyperkit binary (if not in default location)")
	cmd.Flags().StringVar(&data, "data", "", "String of metadata to pass to VM; error to specify both -data and -data-file")
	cmd.Flags().StringVar(&dataPath, "data-file", "", "Path to file containing metadata to pass to VM; error to specify both -data and -data-file")
	cmd.Flags().StringVar(&ipStr, "ip", "", "Preferred IPv4 address for the VM.")
	cmd.Flags().StringVar(&state, "state", "", "Path to directory to keep VM state in")
	cmd.Flags().StringVar(&vsockports, "vsock-ports", "", "List of vsock ports to forward from the guest on startup (comma separated). A unix domain socket for each port will be created in the state directory")
	cmd.Flags().StringVar(&networking, "networking", hyperkitNetworkingDefault, "Networking mode. Valid options are 'default', 'docker-for-mac', 'vpnkit[,eth-socket-path[,port-socket-path]]', 'vmnet' and 'none'. 'docker-for-mac' connects to the network used by Docker for Mac. 'vpnkit' connects to the VPNKit socket(s) specified. If no socket path is provided a new VPNKit instance will be started and 'vpnkit_eth.sock' and 'vpnkit_port.sock' will be created in the state directory. 'port-socket-path' is only needed if you want to publish ports on localhost using an existing VPNKit instance. 'vmnet' uses the Apple vmnet framework, requires root/sudo. 'none' disables networking.`")

	cmd.Flags().StringVar(&vpnkitUUID, "vpnkit-uuid", "", "Optional UUID used to identify the VPNKit connection. Overrides 'vpnkit.uuid' in the state directory.")
	cmd.Flags().StringVar(&vpnkitPath, "vpnkit", "", "Path to vpnkit binary")
	cmd.Flags().Var(&publishFlags, "publish", "Publish a vm's port(s) to the host (default [])")

	// Boot type; we try to determine automatically
	cmd.Flags().BoolVar(&uefiBoot, "uefi", false, "Use UEFI boot")
	cmd.Flags().BoolVar(&isoBoot, "iso", false, "Boot image is an ISO")
	cmd.Flags().BoolVar(&squashFSBoot, "squashfs", false, "Boot image is a kernel+squashfs+cmdline")
	cmd.Flags().BoolVar(&kernelBoot, "kernel", false, "Boot image is kernel+initrd+cmdline 'path'-kernel/-initrd/-cmdline")

	// Hyperkit settings
	cmd.Flags().BoolVar(&consoleToFile, "console-file", false, "Output the console to a tty file")

	// Paths and settings for UEFI firmware
	// Note, the default uses the firmware shipped with Docker for Mac
	cmd.Flags().StringVar(&fw, "fw", "/Applications/Docker.app/Contents/Resources/uefi/UEFI.fd", "Path to OVMF firmware for UEFI boot")

	return cmd
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

	go func() {
		_ = cmd.Wait() // run in background
	}()

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
	vmnetClient, err := vmnet.New(ctx, h.VPNKitSock)
	if err != nil {
		return nil, fmt.Errorf("NewVmnet failed: %v", err)
	}
	defer vmnetClient.Close()

	// Register with VPNKit
	var vif *vmnet.Vif
	if h.VPNKitPreferredIPv4 == "" {
		log.Debugf("Creating VPNKit VIF for %v", vpnkitUUID)
		vif, err = vmnetClient.ConnectVif(vpnkitUUID)
		if err != nil {
			return nil, fmt.Errorf("Connection to Vif failed: %v", err)
		}
	} else {
		ip := net.ParseIP(h.VPNKitPreferredIPv4)
		if ip == nil {
			return nil, fmt.Errorf("Failed to parse IP: %s", h.VPNKitPreferredIPv4)
		}
		log.Debugf("Creating VPNKit VIF for %v ip=%v", vpnkitUUID, ip)
		vif, err = vmnetClient.ConnectVifIP(vpnkitUUID, ip)
		if err != nil {
			return nil, fmt.Errorf("Connection to Vif with IP failed: %v", err)
		}
	}
	log.Debugf("VPNKit UUID:%s IP: %v", vpnkitUUID, vif.IP)

	log.Debugf("Connecting to VPNKit on %s", portSocket)
	c, err := vpnkit.NewClient(portSocket)
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
		vp := &vpnkit.Port{
			Proto:   vpnkit.Protocol(p.Protocol),
			OutIP:   localhost,
			OutPort: p.Host,
			InIP:    vif.IP,
			InPort:  p.Guest,
		}
		if err = c.Expose(context.Background(), vp); err != nil {
			return nil, fmt.Errorf("Failed to expose port %s: %v", publish, err)
		}
		ports = append(ports, vp)
	}

	// Return cleanup function
	return func() {
		for _, vp := range ports {
			_ = c.Unexpose(context.Background(), vp)
		}
	}, nil
}
