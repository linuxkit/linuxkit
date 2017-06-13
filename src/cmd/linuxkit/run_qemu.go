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
const QemuImg = "linuxkit/qemu:c9691f5c50dd191e62b77eaa2f3dfd05ed2ed77c"

// QemuConfig contains the config for Qemu
type QemuConfig struct {
	Path           string
	ISOBoot        bool
	UEFI           bool
	Kernel         bool
	GUI            bool
	Disks          Disks
	MetadataPath   string
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

	// State flags
	state := flags.String("state", "", "Path to directory to keep VM state in")

	// Paths and settings for disks
	var disks Disks
	flags.Var(&disks, "disk", "Disk config, may be repeated. [file=]path[,size=1G][,format=qcow2]")
	data := flags.String("data", "", "Metadata to pass to VM (either a path to a file or a string)")

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
	prefix := path

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
			prefix = strings.TrimSuffix(path, ".iso")
		}
		// autodetect EFI ISO from our default naming
		if strings.HasSuffix(path, "-efi.iso") {
			*uefiBoot = true
			prefix = strings.TrimSuffix(path, "-efi.iso")
		}
	}

	if *state == "" {
		*state = prefix + "-state"
	}
	var mkstate func()
	mkstate = func() {
		if err := os.MkdirAll(*state, 0755); err != nil {
			log.Fatalf("Could not create state directory: %v", err)
		}
		mkstate = func() {}
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
		mkstate()
		isoPath = filepath.Join(*state, "data.iso")
		if err := WriteMetadataISO(isoPath, d); err != nil {
			log.Fatalf("Cannot write user data ISO: %v", err)
		}
	}

	for i, d := range disks {
		id := ""
		if i != 0 {
			id = strconv.Itoa(i)
		}
		if d.Size != 0 && d.Format == "" {
			d.Format = "qcow2"
		}
		if d.Size != 0 && d.Path == "" {
			mkstate()
			d.Path = filepath.Join(*state, "disk"+id+".img")
		}
		if d.Path == "" {
			log.Fatalf("disk specified with no size or name")
		}
		disks[i] = d
	}

	// user not trying to boot off ISO or kernel, so assume booting from a disk image
	if !*kernelBoot && !*isoBoot {
		// currently no way to set format, but autodetect probably works
		d := Disks{DiskConfig{Path: path}}
		disks = append(d, disks...)
	}

	if *isoBoot && isoPath != "" {
		log.Fatalf("metadata and ISO boot currently cannot coexist")
	}

	config := QemuConfig{
		Path:           path,
		ISOBoot:        *isoBoot,
		UEFI:           *uefiBoot,
		Kernel:         *kernelBoot,
		GUI:            *enableGUI,
		Disks:          disks,
		MetadataPath:   isoPath,
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
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	var binds []string
	addBind := func(p string) {
		if filepath.IsAbs(p) {
			binds = append(binds, "-v", fmt.Sprintf("%[1]s:%[1]s", filepath.Dir(p)))
		} else {
			binds = append(binds, "-v", fmt.Sprintf("%[1]s:%[1]s", cwd))
		}
	}
	addBind(config.Path)

	if config.MetadataPath != "" {
		addBind(config.MetadataPath)
	}
	// also try to bind mount disk paths so the command works
	for _, d := range config.Disks {
		addBind(d.Path)
	}

	var args []string
	config, args = buildQemuCmdline(config)

	dockerArgs := append([]string{"run", "--interactive", "--rm", "-w", cwd}, binds...)
	dockerArgsImg := append([]string{"run", "--rm", "-w", cwd}, binds...)

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

	for _, d := range config.Disks {
		// If disk doesn't exist then create one
		if _, err = os.Stat(d.Path); err != nil {
			if os.IsNotExist(err) {
				log.Debugf("Creating new qemu disk [%s] format %s", d.Path, d.Format)
				imgArgs := append(dockerArgsImg, QemuImg, "qemu-img", "create", "-f", d.Format, d.Path, fmt.Sprintf("%dM", d.Size))
				qemuImgCmd := exec.Command(dockerPath, imgArgs...)
				qemuImgCmd.Stderr = os.Stderr
				log.Debugf("%v\n", qemuImgCmd.Args)
				if err = qemuImgCmd.Run(); err != nil {
					return fmt.Errorf("Error creating disk [%s] format %s:  %s", d.Path, d.Format, err.Error())
				}
			} else {
				return err
			}
		} else {
			log.Infof("Using existing disk [%s] format %s", d.Path, d.Format)
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
	}

	if config.ISOBoot {
		qemuArgs = append(qemuArgs, "-cdrom", config.Path)
		qemuArgs = append(qemuArgs, "-boot", "d")
	} else if config.MetadataPath != "" {
		qemuArgs = append(qemuArgs, "-cdrom", config.MetadataPath)
	}

	if config.UEFI {
		qemuArgs = append(qemuArgs, "-pflash", config.FWPath)
	}

	// build kernel boot config from kernel/initrd/cmdline
	if config.Kernel {
		qemuKernelPath := config.Path + "-kernel"
		qemuInitrdPath := config.Path + "-initrd.img"
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
