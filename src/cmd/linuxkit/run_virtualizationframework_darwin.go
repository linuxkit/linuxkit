//go:build darwin
// +build darwin

package main

import (
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/Code-Hex/vz"
	"github.com/pkg/term/termios"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

const (
	virtualizationNetworkingNone         string = "none"
	virtualizationNetworkingDockerForMac        = "docker-for-mac"
	virtualizationNetworkingVPNKit              = "vpnkit"
	virtualizationNetworkingVMNet               = "vmnet"
	virtualizationNetworkingDefault             = virtualizationNetworkingVMNet
	virtualizationFrameworkConsole              = "console=hvc0"
)

// Process the run arguments and execute run
func runVirtualizationFramework(args []string) {
	flags := flag.NewFlagSet("virtualization", flag.ExitOnError)
	invoked := filepath.Base(os.Args[0])
	flags.Usage = func() {
		fmt.Printf("USAGE: %s run virtualization [options] prefix\n\n", invoked)
		fmt.Printf("'prefix' specifies the path to the VM image.\n")
		fmt.Printf("\n")
		fmt.Printf("Options:\n")
		flags.PrintDefaults()
	}
	cpus := flags.Uint("cpus", 1, "Number of CPUs")
	mem := flags.Uint64("mem", 1024, "Amount of memory in MB")
	memBytes := *mem * 1024 * 1024
	var disks Disks
	flags.Var(&disks, "disk", "Disk config. [file=]path[,size=1G]")
	data := flags.String("data", "", "String of metadata to pass to VM; error to specify both -data and -data-file")
	dataPath := flags.String("data-file", "", "Path to file containing metadata to pass to VM; error to specify both -data and -data-file")

	if *data != "" && *dataPath != "" {
		log.Fatal("Cannot specify both -data and -data-file")
	}

	state := flags.String("state", "", "Path to directory to keep VM state in")
	networking := flags.String("networking", virtualizationNetworkingDefault, "Networking mode. Valid options are 'default', 'vmnet' and 'none'. 'vmnet' uses the Apple vmnet framework. 'none' disables networking.`")

	kernelBoot := flags.Bool("kernel", false, "Boot image is kernel+initrd+cmdline 'path'-kernel/-initrd/-cmdline")

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

	metadataPaths, err := CreateMetadataISO(*state, *data, *dataPath)
	if err != nil {
		log.Fatalf("%v", err)
	}
	isoPaths = append(isoPaths, metadataPaths...)

	// TODO: We should generate new UUID, otherwise /sys/class/dmi/id/product_uuid is identical on all VMs.
	// but it is not clear if support is there in VF, or if it is built-in

	// Run

	cmdlineBytes, err := ioutil.ReadFile(prefix + "-cmdline")
	if err != nil {
		log.Fatalf("Cannot open cmdline file: %v", err)
	}
	// must have hvc0 as console for vf
	kernelCommandLineArguments := strings.Split(string(cmdlineBytes), " ")

	// Use the first virtio console device as system console.
	//"console=hvc0",
	// Stop in the initial ramdisk before attempting to transition to
	// the root file system.
	//"root=/dev/vda",
	kernelCommandLineArguments = append(kernelCommandLineArguments, "console=hvc0")

	vmlinuz := prefix + "-kernel"
	initrd := prefix + "-initrd.img"

	vmlinuzFile := vmlinuz
	// need to check if it is gzipped, and, if so, gunzip it
	filetype, err := checkFileType(vmlinuz)
	if err != nil {
		log.Fatalf("unable to check kernel file type at %s: %v", vmlinuz, err)
	}

	if filetype == "application/x-gzip" {
		vmlinuzUncompressed := fmt.Sprintf("%s-uncompressed", vmlinuz)
		// gzipped kernel, we load it into memory, unzip it, and pass it
		f, err := os.Open(vmlinuz)
		if err != nil {
			log.Fatalf("unable to read kernel file %s: %v", vmlinuz, err)
		}
		defer f.Close()
		r, err := gzip.NewReader(f)
		if err != nil {
			log.Fatalf("unable to read from file %s: %v", vmlinuz, err)
		}
		defer r.Close()

		writer, err := os.Create(vmlinuzUncompressed)
		if err != nil {
			log.Fatalf("unable to create decompressed kernel file %s: %v", vmlinuzUncompressed, err)
		}
		defer writer.Close()

		if _, err = io.Copy(writer, r); err != nil {
			log.Fatalf("unable to decompress kernel file to %s: %v", vmlinuzUncompressed, err)
		}
		vmlinuzFile = vmlinuzUncompressed
	}
	bootLoader := vz.NewLinuxBootLoader(
		vmlinuzFile,
		vz.WithCommandLine(strings.Join(kernelCommandLineArguments, " ")),
		vz.WithInitrd(initrd),
	)

	config := vz.NewVirtualMachineConfiguration(
		bootLoader,
		*cpus,
		memBytes,
	)

	// console
	stdin, stdout := os.Stdin, os.Stdout
	serialPortAttachment := vz.NewFileHandleSerialPortAttachment(stdin, stdout)
	consoleConfig := vz.NewVirtioConsoleDeviceSerialPortConfiguration(serialPortAttachment)
	config.SetSerialPortsVirtualMachineConfiguration([]*vz.VirtioConsoleDeviceSerialPortConfiguration{
		consoleConfig,
	})
	setRawMode(os.Stdin)

	// network
	// Select network mode
	// for now, we only support vmnet and none, but hoping to have more in the future
	if *networking == "" || *networking == "default" {
		dflt := virtualizationNetworkingDefault
		networking = &dflt
	}
	netMode := strings.SplitN(*networking, ",", 3)
	switch netMode[0] {

	case virtualizationNetworkingVMNet:
		natAttachment := vz.NewNATNetworkDeviceAttachment()
		networkConfig := vz.NewVirtioNetworkDeviceConfiguration(natAttachment)
		config.SetNetworkDevicesVirtualMachineConfiguration([]*vz.VirtioNetworkDeviceConfiguration{
			networkConfig,
		})
		networkConfig.SetMacAddress(vz.NewRandomLocallyAdministeredMACAddress())
	case virtualizationNetworkingNone:
	default:
		log.Fatalf("Invalid networking mode: %s", netMode[0])
	}

	// entropy
	entropyConfig := vz.NewVirtioEntropyDeviceConfiguration()
	config.SetEntropyDevicesVirtualMachineConfiguration([]*vz.VirtioEntropyDeviceConfiguration{
		entropyConfig,
	})

	var storageDevices []vz.StorageDeviceConfiguration
	for i, d := range disks {
		var id, diskPath string
		if i != 0 {
			id = strconv.Itoa(i)
		}
		if d.Size != 0 && d.Path == "" {
			diskPath = filepath.Join(*state, "disk"+id+".raw")
		}
		if d.Path == "" {
			log.Fatalf("disk specified with no size or name")
		}
		diskImageAttachment, err := vz.NewDiskImageStorageDeviceAttachment(
			diskPath,
			false,
		)
		if err != nil {
			log.Fatal(err)
		}
		storageDeviceConfig := vz.NewVirtioBlockDeviceConfiguration(diskImageAttachment)
		storageDevices = append(storageDevices, storageDeviceConfig)
	}
	for _, iso := range isoPaths {
		diskImageAttachment, err := vz.NewDiskImageStorageDeviceAttachment(
			iso,
			true,
		)
		if err != nil {
			log.Fatal(err)
		}
		storageDeviceConfig := vz.NewVirtioBlockDeviceConfiguration(diskImageAttachment)
		storageDevices = append(storageDevices, storageDeviceConfig)
	}

	config.SetStorageDevicesVirtualMachineConfiguration(storageDevices)

	// traditional memory balloon device which allows for managing guest memory. (optional)
	config.SetMemoryBalloonDevicesVirtualMachineConfiguration([]vz.MemoryBalloonDeviceConfiguration{
		vz.NewVirtioTraditionalMemoryBalloonDeviceConfiguration(),
	})

	// socket device (optional)
	config.SetSocketDevicesVirtualMachineConfiguration([]vz.SocketDeviceConfiguration{
		vz.NewVirtioSocketDeviceConfiguration(),
	})
	validated, err := config.Validate()
	if !validated || err != nil {
		log.Fatal("validation failed", err)
	}

	vm := vz.NewVirtualMachine(config)

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGTERM)

	errCh := make(chan error, 1)

	vm.Start(func(err error) {
		if err != nil {
			errCh <- err
		}
	})

	for {
		select {
		case <-signalCh:
			result, err := vm.RequestStop()
			if err != nil {
				log.Println("request stop error:", err)
				return
			}
			log.Println("recieved signal", result)
		case newState := <-vm.StateChangedNotify():
			if newState == vz.VirtualMachineStateRunning {
				log.Println("start VM is running")
			}
			if newState == vz.VirtualMachineStateStopped {
				log.Println("stopped successfully")
				return
			}
		case err := <-errCh:
			log.Println("in start:", err)
		}
	}

}

// https://developer.apple.com/documentation/virtualization/running_linux_in_a_virtual_machine?language=objc#:~:text=Configure%20the%20Serial%20Port%20Device%20for%20Standard%20In%20and%20Out
func setRawMode(f *os.File) {
	var attr unix.Termios

	// Get settings for terminal
	termios.Tcgetattr(f.Fd(), &attr)

	// Put stdin into raw mode, disabling local echo, input canonicalization,
	// and CR-NL mapping.
	attr.Iflag &^= syscall.ICRNL
	attr.Lflag &^= syscall.ICANON | syscall.ECHO

	// Set minimum characters when reading = 1 char
	attr.Cc[syscall.VMIN] = 1

	// set timeout when reading as non-canonical mode
	attr.Cc[syscall.VTIME] = 0

	// reflects the changed settings
	termios.Tcsetattr(f.Fd(), termios.TCSANOW, &attr)
}

func checkFileType(infile string) (string, error) {
	file, err := os.Open(infile)

	if err != nil {
		return "", err
	}
	defer file.Close()

	b := make([]byte, 512)

	if _, err = file.Read(b); err != nil {
		return "", err
	}

	return http.DetectContentType(b), nil
}
