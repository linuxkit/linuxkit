package main

import (
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	defaultFWPath = "/usr/share/ovmf/bios.bin"
)

// QemuConfig contains the config for Qemu
type QemuConfig struct {
	Path             string
	ISOBoot          bool
	UEFI             bool
	SquashFS         bool
	Kernel           bool
	GUI              bool
	Disks            Disks
	ISOImages        []string
	StatePath        string
	FWPath           string
	Arch             string
	CPUs             string
	Memory           string
	Accel            string
	Detached         bool
	QemuBinPath      string
	QemuImgPath      string
	PublishedPorts   []string
	NetdevConfig     string
	UUID             uuid.UUID
	USB              bool
	Devices          []string
	VirtiofsdBinPath string
	VirtiofsShares   []string
}

const (
	qemuNetworkingNone    string = "none"
	qemuNetworkingUser           = "user"
	qemuNetworkingTap            = "tap"
	qemuNetworkingBridge         = "bridge"
	qemuNetworkingDefault        = qemuNetworkingUser
)

var (
	defaultArch  string
	defaultAccel string
)

func init() {
	switch runtime.GOARCH {
	case "arm64":
		defaultArch = "aarch64"
	case "amd64":
		defaultArch = "x86_64"
	case "s390x":
		defaultArch = "s390x"
	}
	switch {
	case runtime.GOARCH == "s390x":
		defaultAccel = "kvm"
	case haveKVM():
		defaultAccel = "kvm:tcg"
	case runtime.GOOS == "darwin":
		defaultAccel = "hvf:tcg"
	}
}

func haveKVM() bool {
	_, err := os.Stat("/dev/kvm")
	return !os.IsNotExist(err)
}

func retrieveMAC(statePath string) net.HardwareAddr {
	var mac net.HardwareAddr
	fileName := filepath.Join(statePath, "mac-addr")

	if macString, err := os.ReadFile(fileName); err == nil {
		if mac, err = net.ParseMAC(string(macString)); err != nil {
			log.Fatalf("failed to parse mac-addr file: %s\n", macString)
		}
	} else {
		// we did not generate a mac yet. generate one
		mac = generateMAC()
		if err = os.WriteFile(fileName, []byte(mac.String()), 0640); err != nil {
			log.Fatalln("failed to write mac-addr file:", err)
		}
	}

	return mac
}

func generateMAC() net.HardwareAddr {
	mac := make([]byte, 6)
	n, err := rand.Read(mac)
	if err != nil {
		log.WithError(err).Fatal("failed to generate random mac address")
	}
	if n != 6 {
		log.WithError(err).Fatalf("generated %d bytes for random mac address", n)
	}
	mac[0] &^= 0x01 // Clear multicast bit
	mac[0] |= 0x2   // Set locally administered bit
	return mac
}

func runQEMUCmd() *cobra.Command {
	var (
		enableGUI      bool
		uefiBoot       bool
		isoBoot        bool
		squashFSBoot   bool
		kernelBoot     bool
		state          string
		data           string
		dataPath       string
		fw             string
		accel          string
		arch           string
		qemuCmd        string
		qemuDetached   bool
		networking     string
		usbEnabled     bool
		deviceFlags    multipleFlag
		publishFlags   multipleFlag
		virtiofsdCmd   string
		virtiofsShares []string
	)

	cmd := &cobra.Command{
		Use:   "qemu",
		Short: "launch a VM using qemu",
		Long: `Launch a VM using qemu.
		'path' specifies the path to the VM image.

		If not running as root note that '-networking bridge,br0' requires a
		setuid network helper and appropriate host configuration, see
		http://wiki.qemu.org/Features/HelperNetworking.
		`,
		Args:    cobra.ExactArgs(1),
		Example: "linuxkit run qemu [options] path",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]

			if data != "" && dataPath != "" {
				return errors.New("Cannot specify both -data and -data-file")
			}

			// Generate UUID, so that /sys/class/dmi/id/product_uuid is populated
			vmUUID := uuid.New()

			// These envvars override the corresponding command line
			// options. So this must remain after the `flags.Parse` above.
			accel = getStringValue("LINUXKIT_QEMU_ACCEL", accel, "")

			prefix := path

			_, err := os.Stat(path)
			stat := err == nil

			// if the path does not exist, must be trying to do a kernel+initrd or kernel+squashfs boot
			if !stat {
				_, err = os.Stat(path + "-kernel")
				statKernel := err == nil
				if statKernel {
					_, err = os.Stat(path + "-squashfs.img")
					statSquashFS := err == nil
					if statSquashFS {
						squashFSBoot = true
					} else {
						kernelBoot = true
					}
				}
				// we will error out later if neither found
			} else {
				// if path ends in .iso they meant an ISO
				if strings.HasSuffix(path, ".iso") {
					isoBoot = true
					prefix = strings.TrimSuffix(path, ".iso")
				}
			}

			if state == "" {
				state = prefix + "-state"
			}

			if err := os.MkdirAll(state, 0755); err != nil {
				return fmt.Errorf("Could not create state directory: %w", err)
			}

			var isoPaths []string

			if isoBoot {
				isoPaths = append(isoPaths, path)
			}

			metadataPaths, err := CreateMetadataISO(state, data, dataPath)
			if err != nil {
				return err
			}
			isoPaths = append(isoPaths, metadataPaths...)

			for i, d := range disks {
				id := ""
				if i != 0 {
					id = strconv.Itoa(i)
				}
				if d.Size != 0 && d.Format == "" {
					d.Format = "qcow2"
				}
				if d.Size != 0 && d.Path == "" {
					d.Path = filepath.Join(state, "disk"+id+".img")
				}
				if d.Path == "" {
					return fmt.Errorf("disk specified with no size or name")
				}
				disks[i] = d
			}

			// user not trying to boot off ISO or kernel+initrd, so assume booting from a disk image or kernel+squashfs
			if !kernelBoot && !isoBoot {
				var diskPath string
				if squashFSBoot {
					diskPath = path + "-squashfs.img"
				} else {
					if _, err := os.Stat(path); err != nil {
						log.Fatalf("Boot disk image %s does not exist", path)
					}
					diskPath = path
				}
				// currently no way to set format, but autodetect probably works
				d := Disks{DiskConfig{Path: diskPath}}
				disks = append(d, disks...)
			}

			if networking == "" || networking == "default" {
				networking = qemuNetworkingDefault
			}
			netMode := strings.SplitN(networking, ",", 2)

			var netdevConfig string
			switch netMode[0] {
			case qemuNetworkingUser:
				netdevConfig = "user,id=t0"
			case qemuNetworkingTap:
				if len(netMode) != 2 {
					return fmt.Errorf("Not enough arguments for %q networking mode", qemuNetworkingTap)
				}
				if len(publishFlags) != 0 {
					return fmt.Errorf("Port publishing requires %q networking mode", qemuNetworkingUser)
				}
				netdevConfig = fmt.Sprintf("tap,id=t0,ifname=%s,script=no,downscript=no", netMode[1])
			case qemuNetworkingBridge:
				if len(netMode) != 2 {
					return fmt.Errorf("Not enough arguments for %q networking mode", qemuNetworkingBridge)
				}
				if len(publishFlags) != 0 {
					return fmt.Errorf("Port publishing requires %q networking mode", qemuNetworkingUser)
				}
				netdevConfig = fmt.Sprintf("bridge,id=t0,br=%s", netMode[1])
			case qemuNetworkingNone:
				if len(publishFlags) != 0 {
					return fmt.Errorf("Port publishing requires %q networking mode", qemuNetworkingUser)
				}
				netdevConfig = ""
			default:
				return fmt.Errorf("Invalid networking mode: %s", netMode[0])
			}

			config := QemuConfig{
				Path:             path,
				ISOBoot:          isoBoot,
				UEFI:             uefiBoot,
				SquashFS:         squashFSBoot,
				Kernel:           kernelBoot,
				GUI:              enableGUI,
				Disks:            disks,
				ISOImages:        isoPaths,
				StatePath:        state,
				FWPath:           fw,
				Arch:             arch,
				CPUs:             fmt.Sprintf("%d", cpus),
				Memory:           fmt.Sprintf("%d", mem),
				Accel:            accel,
				Detached:         qemuDetached,
				QemuBinPath:      qemuCmd,
				PublishedPorts:   publishFlags,
				NetdevConfig:     netdevConfig,
				UUID:             vmUUID,
				USB:              usbEnabled,
				Devices:          deviceFlags,
				VirtiofsdBinPath: virtiofsdCmd,
				VirtiofsShares:   virtiofsShares,
			}

			config, err = discoverQemu(config)
			if err != nil {
				return err
			}

			if len(config.VirtiofsShares) > 0 {
				config, err = discoverVirtiofsd(config)
				if err != nil {
					return err
				}
			}

			if err = runQemuLocal(config); err != nil {
				return err
			}
			return nil
		},
	}

	// Display flags
	cmd.Flags().BoolVar(&enableGUI, "gui", false, "Set qemu to use video output instead of stdio")

	// Boot type; we try to determine automatically
	cmd.Flags().BoolVar(&uefiBoot, "uefi", false, "Use UEFI boot")
	cmd.Flags().BoolVar(&isoBoot, "iso", false, "Boot image is an ISO")
	cmd.Flags().BoolVar(&squashFSBoot, "squashfs", false, "Boot image is a kernel+squashfs+cmdline")
	cmd.Flags().BoolVar(&kernelBoot, "kernel", false, "Boot image is kernel+initrd+cmdline 'path'-kernel/-initrd/-cmdline")

	// State flags
	cmd.Flags().StringVar(&state, "state", "", "Path to directory to keep VM state in")

	// Paths and settings for disks
	cmd.Flags().StringVar(&data, "data", "", "String of metadata to pass to VM; error to specify both -data and -data-file")
	cmd.Flags().StringVar(&dataPath, "data-file", "", "Path to file containing metadata to pass to VM; error to specify both -data and -data-file")

	// Paths and settings for UEFI firware
	// Note, we do not use defaultFWPath here as we have a special case for containerised execution
	cmd.Flags().StringVar(&fw, "fw", "", "Path to OVMF firmware for UEFI boot")

	// VM configuration
	cmd.Flags().StringVar(&accel, "accel", defaultAccel, "Choose acceleration mode. Use 'tcg' to disable it.")
	cmd.Flags().StringVar(&arch, "arch", defaultArch, "Type of architecture to use, e.g. x86_64, aarch64, s390x")

	// Backend configuration
	cmd.Flags().StringVar(&qemuCmd, "qemu", "", "Path to the qemu binary (otherwise look in $PATH)")
	cmd.Flags().BoolVar(&qemuDetached, "detached", false, "Set qemu container to run in the background")

	// Networking
	cmd.Flags().StringVar(&networking, "networking", qemuNetworkingDefault, "Networking mode. Valid options are 'default', 'user', 'bridge[,name]', tap[,name] and 'none'. 'user' uses QEMUs userspace networking. 'bridge' connects to a preexisting bridge. 'tap' uses a prexisting tap device. 'none' disables networking.`")

	cmd.Flags().Var(&publishFlags, "publish", "Publish a vm's port(s) to the host (default [])")

	// USB devices
	cmd.Flags().BoolVar(&usbEnabled, "usb", false, "Enable USB controller")
	cmd.Flags().Var(&deviceFlags, "device", "Add USB host device(s). Format driver[,prop=value][,...] -- add device, like -device on the qemu command line.")

	// Filesystems
	cmd.Flags().StringVar(&virtiofsdCmd, "virtiofsd", "", "Path to virtiofsd binary (otherwise look in /usr/lib/qemu and /usr/local/lib/qemu)")
	cmd.Flags().StringArrayVar(&virtiofsShares, "virtiofs", []string{}, "Directory shared on virtiofs")

	return cmd
}

func runQemuLocal(config QemuConfig) error {
	var args []string
	config, args = buildQemuCmdline(config)

	for _, d := range config.Disks {
		// If disk doesn't exist then create one
		if _, err := os.Stat(d.Path); err != nil {
			if os.IsNotExist(err) {
				log.Debugf("Creating new qemu disk [%s] format %s", d.Path, d.Format)
				qemuImgCmd := exec.Command(config.QemuImgPath, "create", "-f", d.Format, d.Path, fmt.Sprintf("%dM", d.Size))
				log.Debugf("%v\n", qemuImgCmd.Args)
				if err := qemuImgCmd.Run(); err != nil {
					return fmt.Errorf("Error creating disk [%s] format %s:  %s", d.Path, d.Format, err.Error())
				}
			} else {
				return err
			}
		} else {
			log.Infof("Using existing disk [%s] format %s", d.Path, d.Format)
		}
	}

	// Check for OVMF firmware before running
	if config.UEFI {
		if config.FWPath == "" {
			// there is no default on mac
			if runtime.GOOS == "darwin" {
				return fmt.Errorf("To run qemu with UEFI firmware on macOS, you must specify the path to locally installed OVMF firmware as `--fw <path>`. You can download OVMF from https://sourceforge.net/projects/edk2/files/OVMF/ ")
			}
			config.FWPath = defaultFWPath
		}
		if _, err := os.Stat(config.FWPath); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("File [%s] does not exist, please ensure OVMF is installed", config.FWPath)
			}
			return err
		}
	}

	// Detached mode is only supported in a container.
	if config.Detached {
		return fmt.Errorf("Detached mode is only supported when running in a container, not locally")
	}

	if len(config.VirtiofsShares) > 0 {
		args = append(args, "-object", "memory-backend-memfd,id=mem,size="+config.Memory+"M,share=on", "-numa", "node,memdev=mem")
	}
	for index, source := range config.VirtiofsShares {
		socket := filepath.Join(config.StatePath, fmt.Sprintf("%s%d", "virtiofs", index))

		cmd := exec.Command(config.VirtiofsdBinPath,
			"--socket-path="+socket,
			"-o", fmt.Sprintf("source=%s", source))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("virtiofs server cannot start: %v", err)
		}
		args = append(args, "-chardev", fmt.Sprintf("socket,id=char%d,path=%s", index, socket))
		args = append(args, "-device",
			fmt.Sprintf("vhost-user-fs-pci,chardev=char%d,tag=virtiofs%d", index, index))
	}

	qemuCmd := exec.Command(config.QemuBinPath, args...)
	// If verbosity is enabled print out the full path/arguments
	log.Debugf("%v\n", qemuCmd.Args)

	// If we're not using a separate window then link the execution to stdin/out
	if !config.GUI {
		qemuCmd.Stdin = os.Stdin
		qemuCmd.Stdout = os.Stdout
		qemuCmd.Stderr = os.Stderr
	}

	return qemuCmd.Run()
}

func buildQemuCmdline(config QemuConfig) (QemuConfig, []string) {
	// Iterate through the flags and build arguments
	var qemuArgs []string
	qemuArgs = append(qemuArgs, "-smp", config.CPUs)
	qemuArgs = append(qemuArgs, "-m", config.Memory)
	qemuArgs = append(qemuArgs, "-uuid", config.UUID.String())
	qemuArgs = append(qemuArgs, "-pidfile", filepath.Join(config.StatePath, "qemu.pid"))

	// Need to specify the vcpu type when running qemu on arm64 platform, for security reason,
	// the vcpu should be "host" instead of other names such as "cortex-a53"...
	if config.Arch == "aarch64" {
		if runtime.GOARCH == "arm64" {
			qemuArgs = append(qemuArgs, "-cpu", "host")
		} else {
			qemuArgs = append(qemuArgs, "-cpu", "cortex-a57")
		}
	}

	// goArch is the GOARCH equivalent of config.Arch
	var goArch string
	switch config.Arch {
	case "s390x":
		goArch = "s390x"
	case "aarch64":
		goArch = "arm64"
	case "x86_64":
		goArch = "amd64"
	default:
		log.Fatalf("%s is an unsupported architecture.", config.Arch)
	}

	if goArch != runtime.GOARCH {
		log.Infof("Disable acceleration as %s != %s", config.Arch, runtime.GOARCH)
		config.Accel = ""
	}

	if config.Accel != "" {
		switch config.Arch {
		case "s390x":
			qemuArgs = append(qemuArgs, "-machine", fmt.Sprintf("s390-ccw-virtio,accel=%s", config.Accel))
		case "aarch64":
			gic := ""
			// VCPU supports less PA bits (36) than requested by the memory map (40)
			highmem := "highmem=off,"
			if runtime.GOOS == "linux" {
				// gic-version=host requires KVM, which implies Linux
				gic = "gic_version=host,"
				highmem = ""
			}
			qemuArgs = append(qemuArgs, "-machine", fmt.Sprintf("virt,%s%saccel=%s", gic, highmem, config.Accel))
		default:
			qemuArgs = append(qemuArgs, "-machine", fmt.Sprintf("q35,accel=%s", config.Accel))
		}
	} else {
		switch config.Arch {
		case "s390x":
			qemuArgs = append(qemuArgs, "-machine", "s390-ccw-virtio")
		case "aarch64":
			qemuArgs = append(qemuArgs, "-machine", "virt")
		default:
			qemuArgs = append(qemuArgs, "-machine", "q35")
		}
	}

	// rng-random does not work on macOS
	// Temporarily disable it until fixed upstream.
	if runtime.GOOS != "darwin" {
		rng := "rng-random,id=rng0"
		if runtime.GOOS == "linux" {
			rng = rng + ",filename=/dev/urandom"
		}
		if config.Arch == "s390x" {
			qemuArgs = append(qemuArgs, "-object", rng, "-device", "virtio-rng-ccw,rng=rng0")
		} else {
			qemuArgs = append(qemuArgs, "-object", rng, "-device", "virtio-rng-pci,rng=rng0")
		}
	}

	var lastDisk int
	for i, d := range config.Disks {
		index := i
		// hdc is CDROM in qemu
		if i >= 2 && config.ISOBoot {
			index++
		}
		if d.Format != "" {
			qemuArgs = append(qemuArgs, "-drive", "file="+d.Path+",format="+d.Format+",index="+strconv.Itoa(index)+",media=disk")
		} else {
			qemuArgs = append(qemuArgs, "-drive", "file="+d.Path+",index="+strconv.Itoa(index)+",media=disk")
		}
		lastDisk = index
	}

	if config.ISOBoot {
		qemuArgs = append(qemuArgs, "-boot", "d")
	}

	// Ensure CDROMs start from at least hdc
	if lastDisk < 2 {
		lastDisk = 2
	}
	for i, p := range config.ISOImages {
		if i == 0 {
			// This is hdc/CDROM which is skipped by the disk loop above
			if runtime.GOARCH == "s390x" {
				qemuArgs = append(qemuArgs, "-device", "virtio-scsi-ccw")
				qemuArgs = append(qemuArgs, "-device", "scsi-cd,drive=cd1")
				qemuArgs = append(qemuArgs, "-drive", "file="+p+",format=raw,if=none,id=cd1")
			} else {
				qemuArgs = append(qemuArgs, "-cdrom", p)
			}
		} else {
			index := lastDisk + i
			qemuArgs = append(qemuArgs, "-drive", "file="+p+",index="+strconv.Itoa(index)+",media=cdrom")
		}
	}

	if config.UEFI {
		qemuArgs = append(qemuArgs, "-drive", "if=pflash,format=raw,file="+config.FWPath)
	}

	// build kernel boot config from kernel/initrd/cmdline
	switch {
	case config.Kernel:
		qemuKernelPath := config.Path + "-kernel"
		qemuInitrdPath := config.Path + "-initrd.img"
		qemuArgs = append(qemuArgs, "-kernel", qemuKernelPath)
		qemuArgs = append(qemuArgs, "-initrd", qemuInitrdPath)
		cmdlineBytes, err := os.ReadFile(config.Path + "-cmdline")
		if err != nil {
			log.Errorf("Cannot open cmdline file: %v", err)
		} else {
			qemuArgs = append(qemuArgs, "-append", string(cmdlineBytes))
		}
	case config.SquashFS:
		qemuKernelPath := config.Path + "-kernel"
		qemuArgs = append(qemuArgs, "-kernel", qemuKernelPath)
		cmdlineBytes, err := os.ReadFile(config.Path + "-cmdline")
		if err != nil {
			log.Errorf("Cannot open cmdline file: %v", err)
		} else {
			cmdline := string(cmdlineBytes)
			cmdline += " root=/dev/sda"
			qemuArgs = append(qemuArgs, "-append", cmdline)
		}
	}

	if config.NetdevConfig == "" {
		qemuArgs = append(qemuArgs, "-net", "none")
	} else {
		mac := retrieveMAC(config.StatePath)
		if config.Arch == "s390x" {
			qemuArgs = append(qemuArgs, "-device", "virtio-net-ccw,netdev=t0,mac="+mac.String())
		} else {
			qemuArgs = append(qemuArgs, "-device", "virtio-net-pci,netdev=t0,mac="+mac.String())
		}
		forwardings, err := buildQemuForwardings(config.PublishedPorts)
		if err != nil {
			log.Error(err)
		}
		qemuArgs = append(qemuArgs, "-netdev", config.NetdevConfig+forwardings)
	}

	if !config.GUI {
		qemuArgs = append(qemuArgs, "-nographic")
	}

	if config.USB {
		qemuArgs = append(qemuArgs, "-usb")
	}
	for _, d := range config.Devices {
		qemuArgs = append(qemuArgs, "-device", d)
	}

	return config, qemuArgs
}

func discoverQemu(config QemuConfig) (QemuConfig, error) {
	if config.QemuImgPath != "" {
		return config, nil
	}

	qemuBinPath := "qemu-system-" + config.Arch
	qemuImgPath := "qemu-img"

	var err error
	config.QemuBinPath, err = exec.LookPath(qemuBinPath)
	if err != nil {
		return config, fmt.Errorf("Unable to find %s within the $PATH", qemuBinPath)
	}

	config.QemuImgPath, err = exec.LookPath(qemuImgPath)
	if err != nil {
		return config, fmt.Errorf("Unable to find %s within the $PATH", qemuImgPath)
	}

	return config, nil
}

func discoverVirtiofsd(config QemuConfig) (QemuConfig, error) {
	if config.VirtiofsdBinPath != "" {
		return config, nil
	}

	virtiofsdPath := filepath.Dir(config.QemuBinPath)
	config.VirtiofsdBinPath = filepath.Join(virtiofsdPath, "..", "lib", "qemu", "virtiofsd")
	return config, nil
}

func buildQemuForwardings(publishFlags multipleFlag) (string, error) {
	if len(publishFlags) == 0 {
		return "", nil
	}
	var forwardings string
	for _, publish := range publishFlags {
		p, err := NewPublishedPort(publish)
		if err != nil {
			return "", err
		}

		hostPort := p.Host
		guestPort := p.Guest

		forwardings = fmt.Sprintf("%s,hostfwd=%s::%d-:%d", forwardings, p.Protocol, hostPort, guestPort)
	}

	return forwardings, nil
}
