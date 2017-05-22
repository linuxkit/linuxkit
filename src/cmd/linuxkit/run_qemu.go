package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"
)

// QemuImg is the version of qemu container
const QemuImg = "linuxkit/qemu:17f052263d63c8a2b641ad91c589edcbb8a18c82"

// QemuConfig contains the config for Qemu
type QemuConfig struct {
	Path           string
	ISO            bool
	UEFI           bool
	Kernel         bool
	GUI            bool
	DiskPath       string
	DiskSize       string
	DiskFormat     string
	FWPath         string
	Arch           string
	CPUs           string
	Memory         string
	KVM            bool
	Containerized  bool
	QemuBinPath    string
	QemuImgPath    string
	PublishedPorts []string
}

func runQemu(args []string) {
	invoked := filepath.Base(os.Args[0])
	flags := flag.NewFlagSet("qemu", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Printf("USAGE: %s run qemu [options] path\n\n", invoked)
		fmt.Printf("'path' specifies the path to the VM image.\n")
		fmt.Printf("\n")
		fmt.Printf("Options:\n")
		flags.PrintDefaults()
	}

	// Display flags
	enableGUI := flags.Bool("gui", false, "Set qemu to use video output instead of stdio")

	// Boot type; we try to determine automatically
	uefiBoot := flags.Bool("uefi", false, "Use UEFI boot")
	isoBoot := flags.Bool("iso", false, "Boot image is an ISO")
	kernelBoot := flags.Bool("kernel", false, "Boot image is kernel+initrd+cmdline 'path'-kernel/-initrd/-cmdline")

	// Paths and settings for disks
	disk := flags.String("disk", "", "Path to disk image to use")
	diskSz := flags.String("disk-size", "", "Size of disk to create, only created if it doesn't exist")
	diskFmt := flags.String("disk-format", "qcow2", "Format of disk: raw, qcow2 etc")

	// Paths and settings for UEFI firware
	fw := flags.String("fw", "/usr/share/ovmf/bios.bin", "Path to OVMF firmware for UEFI boot")

	// VM configuration
	arch := flags.String("arch", "x86_64", "Type of architecture to use, e.g. x86_64, aarch64")
	cpus := flags.String("cpus", "1", "Number of CPUs")
	mem := flags.String("mem", "1024", "Amount of memory in MB")

	// Backend configuration
	qemuContainerized := flags.Bool("containerized", false, "Run qemu in a container")

	publishFlags := multipleFlag{}
	flags.Var(&publishFlags, "publish", "Publish a vm's port(s) to the host (default [])")

	if err := flags.Parse(args); err != nil {
		log.Fatal("Unable to parse args")
	}
	remArgs := flags.Args()

	if len(remArgs) == 0 {
		fmt.Println("Please specify the path to the image to boot")
		flags.Usage()
		os.Exit(1)
	}
	path := remArgs[0]

	_, err := os.Stat(path)
	stat := err == nil

	// if the path does not exist, must be trying to do a kernel boot
	if !stat {
		_, err = os.Stat(path + "-kernel")
		statKernel := err == nil
		if statKernel {
			*kernelBoot = true
		}
		// we will error out later if neither found
	} else {
		// if path ends in .iso they meant an ISO
		if strings.HasSuffix(path, ".iso") {
			*isoBoot = true
		}
		// autodetect EFI ISO from our default naming
		if strings.HasSuffix(path, "-efi.iso") {
			*uefiBoot = true
		}
	}

	// user not trying to boot off ISO or kernel, so assume booting from a disk image
	if !*kernelBoot && !*isoBoot {
		if *disk != "" {
			// Need to add multiple disk support to do this
			log.Fatalf("Cannot boot from disk and specify a disk as well at present")
		}
		*disk = path
	}

	config := QemuConfig{
		Path:           path,
		ISO:            *isoBoot,
		UEFI:           *uefiBoot,
		Kernel:         *kernelBoot,
		GUI:            *enableGUI,
		DiskPath:       *disk,
		DiskSize:       *diskSz,
		DiskFormat:     *diskFmt,
		FWPath:         *fw,
		Arch:           *arch,
		CPUs:           *cpus,
		Memory:         *mem,
		Containerized:  *qemuContainerized,
		PublishedPorts: publishFlags,
	}

	config = discoverBackend(config)

	if config.Containerized {
		err = runQemuContainer(config)
	} else {
		err = runQemuLocal(config)
	}
	if err != nil {
		log.Fatal(err.Error())
	}
}

func runQemuLocal(config QemuConfig) error {
	var args []string
	config, args = buildQemuCmdline(config)

	if config.DiskPath != "" {
		// If disk doesn't exist then create one
		if _, err := os.Stat(config.DiskPath); err != nil {
			if os.IsNotExist(err) {
				log.Infof("Creating new qemu disk [%s] format %s", config.DiskPath, config.DiskFormat)
				qemuImgCmd := exec.Command(config.QemuImgPath, "create", "-f", config.DiskFormat, config.DiskPath, config.DiskSize)
				log.Debugf("%v\n", qemuImgCmd.Args)
				if err := qemuImgCmd.Run(); err != nil {
					return fmt.Errorf("Error creating disk [%s] format %s:  %s", config.DiskPath, config.DiskFormat, err.Error())
				}
			} else {
				return err
			}
		} else {
			log.Infof("Using existing disk [%s] format %s", config.DiskPath, config.DiskFormat)
		}
	}

	// Check for OVMF firmware before running
	if config.UEFI {
		if _, err := os.Stat(config.FWPath); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("File [%s] does not exist, please ensure OVMF is installed", config.FWPath)
			}
			return err
		}
	}

	qemuCmd := exec.Command(config.QemuBinPath, args...)
	// If verbosity is enabled print out the full path/arguments
	log.Debugf("%v\n", qemuCmd.Args)

	// If we're not using a separate window then link the execution to stdin/out
	if config.GUI != true {
		qemuCmd.Stdin = os.Stdin
		qemuCmd.Stdout = os.Stdout
		qemuCmd.Stderr = os.Stderr
	}

	return qemuCmd.Run()
}

func runQemuContainer(config QemuConfig) error {
	var wd string
	if filepath.IsAbs(config.Path) {
		// Split the path
		wd, config.Path = filepath.Split(config.Path)
		log.Debugf("Path: %s", config.Path)
	} else {
		var err error
		wd, err = os.Getwd()
		if err != nil {
			return err
		}
	}

	var args []string
	config, args = buildQemuCmdline(config)

	dockerArgs := []string{"run", "--interactive", "--rm", "-v", fmt.Sprintf("%s:%s", wd, "/tmp"), "-w", "/tmp"}
	dockerArgsImg := []string{"run", "--rm", "-v", fmt.Sprintf("%s:%s", wd, "/tmp"), "-w", "/tmp"}

	if terminal.IsTerminal(int(os.Stdin.Fd())) {
		dockerArgs = append(dockerArgs, "--tty")
	}

	if config.KVM {
		dockerArgs = append(dockerArgs, "--device", "/dev/kvm")
	}

	if config.PublishedPorts != nil && len(config.PublishedPorts) > 0 {
		forwardings, err := buildDockerForwardings(config.PublishedPorts)
		if err != nil {
			return err
		}
		dockerArgs = append(dockerArgs, forwardings...)
	}

	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		return fmt.Errorf("Unable to find docker in the $PATH")
	}

	if config.DiskPath != "" {
		// If disk doesn't exist then create one
		if _, err = os.Stat(config.DiskPath); err != nil {
			if os.IsNotExist(err) {
				log.Infof("Creating new qemu disk [%s] format %s", config.DiskPath, config.DiskFormat)
				imgArgs := append(dockerArgsImg, QemuImg, "qemu-img", "create", "-f", config.DiskFormat, config.DiskPath, config.DiskSize)
				qemuImgCmd := exec.Command(dockerPath, imgArgs...)
				qemuImgCmd.Stderr = os.Stderr
				log.Debugf("%v\n", qemuImgCmd.Args)
				if err = qemuImgCmd.Run(); err != nil {
					return fmt.Errorf("Error creating disk [%s] format %s:  %s", config.DiskPath, config.DiskFormat, err.Error())
				}
			} else {
				return err
			}
		} else {
			log.Infof("Using existing disk [%s] format %s", config.DiskPath, config.DiskFormat)
		}
	}

	qemuArgs := append(dockerArgs, QemuImg, "qemu-system-"+config.Arch)
	qemuArgs = append(qemuArgs, args...)
	qemuCmd := exec.Command(dockerPath, qemuArgs...)
	// If verbosity is enabled print out the full path/arguments
	log.Debugf("%v\n", qemuCmd.Args)

	// GUI mode not currently supported in a container. Although it could be in future.
	if config.GUI == true {
		return fmt.Errorf("GUI mode is only supported when running locally, not in a container")
	}

	qemuCmd.Stdin = os.Stdin
	qemuCmd.Stdout = os.Stdout
	qemuCmd.Stderr = os.Stderr

	return qemuCmd.Run()
}

func buildQemuCmdline(config QemuConfig) (QemuConfig, []string) {
	// Iterate through the flags and build arguments
	var qemuArgs []string
	qemuArgs = append(qemuArgs, "-device", "virtio-rng-pci")
	qemuArgs = append(qemuArgs, "-smp", config.CPUs)
	qemuArgs = append(qemuArgs, "-m", config.Memory)

	// Look for kvm device and enable for qemu if it exists
	var err error
	if _, err = os.Stat("/dev/kvm"); os.IsNotExist(err) {
		qemuArgs = append(qemuArgs, "-machine", "q35")
	} else {
		config.KVM = true
		qemuArgs = append(qemuArgs, "-enable-kvm")
		qemuArgs = append(qemuArgs, "-machine", "q35,accel=kvm:tcg")
	}

	if config.DiskPath != "" {
		qemuArgs = append(qemuArgs, "-drive", "file="+config.DiskPath+",format="+config.DiskFormat+",index=0,media=disk")
	}

	if config.ISO {
		qemuArgs = append(qemuArgs, "-cdrom", config.Path)
		qemuArgs = append(qemuArgs, "-boot", "d")
	}

	if config.UEFI {
		qemuArgs = append(qemuArgs, "-pflash", config.FWPath)
	}

	// build kernel boot config from kernel/initrd/cmdline
	if config.Kernel {
		qemuKernelPath := buildPath(config.Path, "-kernel")
		qemuInitrdPath := buildPath(config.Path, "-initrd.img")
		qemuArgs = append(qemuArgs, "-kernel", qemuKernelPath)
		qemuArgs = append(qemuArgs, "-initrd", qemuInitrdPath)
		cmdlineString, err := ioutil.ReadFile(config.Path + "-cmdline")
		if err != nil {
			log.Errorf("Cannot open cmdline file: %v", err)
		} else {
			qemuArgs = append(qemuArgs, "-append", string(cmdlineString))
		}
	}

	if config.PublishedPorts != nil && len(config.PublishedPorts) > 0 {
		forwardings, err := buildQemuForwardings(config.PublishedPorts, config.Containerized)
		if err != nil {
			log.Error(err)
		}
		qemuArgs = append(qemuArgs, "-net", forwardings)
		qemuArgs = append(qemuArgs, "-net", "nic")
	}

	if config.GUI != true {
		qemuArgs = append(qemuArgs, "-nographic")
	}

	return config, qemuArgs
}

func discoverBackend(config QemuConfig) QemuConfig {
	qemuBinPath := "qemu-system-" + config.Arch
	qemuImgPath := "qemu-img"

	var err error
	config.QemuBinPath, err = exec.LookPath(qemuBinPath)
	if err != nil {
		log.Infof("Unable to find %s within the $PATH. Using a container", qemuBinPath)
		config.Containerized = true
	}

	config.QemuImgPath, err = exec.LookPath(qemuImgPath)
	if err != nil {
		// No need to show the error message twice
		if !config.Containerized {
			log.Infof("Unable to find %s within the $PATH. Using a container", qemuImgPath)
			config.Containerized = true
		}
	}
	return config
}

func buildPath(prefix string, postfix string) string {
	path := prefix + postfix
	if filepath.IsAbs(path) {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			log.Fatalf("File [%s] does not exist in current directory", path)
		}
	}
	return path
}

type multipleFlag []string

type publishedPorts struct {
	guest    int
	host     int
	protocol string
}

func (f *multipleFlag) String() string {
	return "A multiple flag is a type of flag that can be repeated any number of times"
}

func (f *multipleFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}

func splitPublish(publish string) (publishedPorts, error) {
	p := publishedPorts{}
	slice := strings.Split(publish, ":")

	if len(slice) < 2 {
		return p, fmt.Errorf("Unable to parse the ports to be published, should be in format <host>:<guest> or <host>:<guest>/<tcp|udp>")
	}

	hostPort, err := strconv.Atoi(slice[0])

	if err != nil {
		return p, fmt.Errorf("The provided hostPort can't be converted to int")
	}

	right := strings.Split(slice[1], "/")

	protocol := "tcp"
	if len(right) == 2 {
		protocol = strings.TrimSpace(strings.ToLower(right[1]))
	}

	if protocol != "tcp" && protocol != "udp" {
		return p, fmt.Errorf("Provided protocol is not valid, valid options are: udp and tcp")
	}
	guestPort, err := strconv.Atoi(right[0])

	if err != nil {
		return p, fmt.Errorf("The provided guestPort can't be converted to int")
	}

	if hostPort < 1 || hostPort > 65535 {
		return p, fmt.Errorf("Invalid hostPort: %d", hostPort)
	}

	if guestPort < 1 || guestPort > 65535 {
		return p, fmt.Errorf("Invalid guestPort: %d", guestPort)
	}

	p.guest = guestPort
	p.host = hostPort
	p.protocol = protocol
	return p, nil
}

func buildQemuForwardings(publishFlags multipleFlag, containerized bool) (string, error) {
	forwardings := "user"
	for _, publish := range publishFlags {
		p, err := splitPublish(publish)
		if err != nil {
			return "", err
		}

		hostPort := p.host
		guestPort := p.guest

		if containerized {
			hostPort = guestPort
		}
		forwardings = fmt.Sprintf("%s,hostfwd=%s::%d-:%d", forwardings, p.protocol, hostPort, guestPort)
	}

	return forwardings, nil
}

func buildDockerForwardings(publishedPorts []string) ([]string, error) {
	pmap := []string{}
	for _, port := range publishedPorts {
		s, err := splitPublish(port)
		if err != nil {
			return nil, err
		}
		pmap = append(pmap, "-p", fmt.Sprintf("%d:%d/%s", s.host, s.guest, s.protocol))
	}
	return pmap, nil
}
