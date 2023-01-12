//go:build darwin && cgo
// +build darwin,cgo

package main

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	vz "github.com/Code-Hex/vz/v3"
	"github.com/pkg/term/termios"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

// Process the run arguments and execute run
func runVirtualizationFramework(cfg virtualizationFramwworkConfig, path string) error {
	if cfg.data != "" && cfg.dataPath != "" {
		return errors.New("Cannot specify both -data and -data-file")
	}

	prefix := path

	_, err := os.Stat(path + "-kernel")
	statKernel := err == nil

	var isoPaths []string

	// Default to kernel+initrd
	if !statKernel {
		return fmt.Errorf("Cannot find kernel file: %s", path+"-kernel")
	}
	_, err = os.Stat(path + "-initrd.img")
	statInitrd := err == nil
	if !statInitrd {
		return fmt.Errorf("Cannot find initrd file (%s): %w", path+"-initrd.img", err)
	}
	cfg.kernelBoot = true

	metadataPaths, err := CreateMetadataISO(cfg.state, cfg.data, cfg.dataPath)
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	isoPaths = append(isoPaths, metadataPaths...)

	// TODO: We should generate new UUID, otherwise /sys/class/dmi/id/product_uuid is identical on all VMs.
	// but it is not clear if support is there in VF, or if it is built-in

	// Run

	cmdlineBytes, err := os.ReadFile(prefix + "-cmdline")
	if err != nil {
		return fmt.Errorf("Cannot open cmdline file: %v", err)
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
		return fmt.Errorf("unable to check kernel file type at %s: %v", vmlinuz, err)
	}

	if filetype == "application/x-gzip" {
		vmlinuzUncompressed := fmt.Sprintf("%s-uncompressed", vmlinuz)
		// gzipped kernel, we load it into memory, unzip it, and pass it
		f, err := os.Open(vmlinuz)
		if err != nil {
			return fmt.Errorf("unable to read kernel file %s: %v", vmlinuz, err)
		}
		defer f.Close()
		r, err := gzip.NewReader(f)
		if err != nil {
			return fmt.Errorf("unable to read from file %s: %v", vmlinuz, err)
		}
		defer r.Close()

		writer, err := os.Create(vmlinuzUncompressed)
		if err != nil {
			return fmt.Errorf("unable to create decompressed kernel file %s: %v", vmlinuzUncompressed, err)
		}
		defer writer.Close()

		if _, err = io.Copy(writer, r); err != nil {
			return fmt.Errorf("unable to decompress kernel file to %s: %v", vmlinuzUncompressed, err)
		}
		vmlinuzFile = vmlinuzUncompressed
	}
	bootLoader, err := vz.NewLinuxBootLoader(
		vmlinuzFile,
		vz.WithCommandLine(strings.Join(kernelCommandLineArguments, " ")),
		vz.WithInitrd(initrd),
	)
	if err != nil {
		return fmt.Errorf("unable to create bootloader: %v", err)
	}

	config, err := vz.NewVirtualMachineConfiguration(
		bootLoader,
		cfg.cpus,
		cfg.mem,
	)
	if err != nil {
		return fmt.Errorf("unable to create VM config: %v", err)
	}

	// console
	stdin, stdout := os.Stdin, os.Stdout
	serialPortAttachment, err := vz.NewFileHandleSerialPortAttachment(stdin, stdout)
	if err != nil {
		return fmt.Errorf("unable to create serial port attachment: %v", err)
	}
	consoleConfig, err := vz.NewVirtioConsoleDeviceSerialPortConfiguration(serialPortAttachment)
	if err != nil {
		return fmt.Errorf("unable to create console config: %v", err)
	}
	config.SetSerialPortsVirtualMachineConfiguration([]*vz.VirtioConsoleDeviceSerialPortConfiguration{
		consoleConfig,
	})
	setRawMode(os.Stdin)

	// network
	// Select network mode
	// for now, we only support vmnet and none, but hoping to have more in the future
	if cfg.networking == "" || cfg.networking == "default" {
		cfg.networking = virtualizationNetworkingDefault
	}
	netMode := strings.SplitN(cfg.networking, ",", 3)
	switch netMode[0] {

	case virtualizationNetworkingVMNet:
		natAttachment, err := vz.NewNATNetworkDeviceAttachment()
		if err != nil {
			return fmt.Errorf("Could not create NAT network device attachment: %v", err)
		}
		networkConfig, err := vz.NewVirtioNetworkDeviceConfiguration(natAttachment)
		if err != nil {
			return fmt.Errorf("Could not create virtio network device configuration: %v", err)
		}
		config.SetNetworkDevicesVirtualMachineConfiguration([]*vz.VirtioNetworkDeviceConfiguration{
			networkConfig,
		})
		macAddress, err := vz.NewRandomLocallyAdministeredMACAddress()
		if err != nil {
			return fmt.Errorf("Could not create random MAC address: %v", err)
		}
		networkConfig.SetMACAddress(macAddress)
	case virtualizationNetworkingNone:
	default:
		return fmt.Errorf("Invalid networking mode: %s", netMode[0])
	}

	// entropy
	entropyConfig, err := vz.NewVirtioEntropyDeviceConfiguration()
	if err != nil {
		return fmt.Errorf("Could not create virtio entropy device configuration: %v", err)
	}

	config.SetEntropyDevicesVirtualMachineConfiguration([]*vz.VirtioEntropyDeviceConfiguration{
		entropyConfig,
	})

	var storageDevices []vz.StorageDeviceConfiguration
	for i, d := range cfg.disks {
		var id, diskPath string
		if i != 0 {
			id = strconv.Itoa(i)
		}
		if d.Size != 0 && d.Path == "" {
			diskPath = filepath.Join(cfg.state, "disk"+id+".raw")
		}
		if d.Path == "" {
			return fmt.Errorf("disk specified with no size or name")
		}
		diskImageAttachment, err := vz.NewDiskImageStorageDeviceAttachment(
			diskPath,
			false,
		)
		if err != nil {
			log.Fatal(err)
		}
		storageDeviceConfig, err := vz.NewVirtioBlockDeviceConfiguration(diskImageAttachment)
		if err != nil {
			return fmt.Errorf("Could not create virtio block device configuration: %v", err)
		}
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
		storageDeviceConfig, err := vz.NewVirtioBlockDeviceConfiguration(diskImageAttachment)
		if err != nil {
			return fmt.Errorf("Could not create virtio block device configuration: %v", err)
		}
		storageDevices = append(storageDevices, storageDeviceConfig)
	}

	config.SetStorageDevicesVirtualMachineConfiguration(storageDevices)

	// traditional memory balloon device which allows for managing guest memory. (optional)
	memoryBalloonDeviceConfiguration, err := vz.NewVirtioTraditionalMemoryBalloonDeviceConfiguration()
	if err != nil {
		return fmt.Errorf("Could not create virtio traditional memory balloon device configuration: %v", err)
	}
	config.SetMemoryBalloonDevicesVirtualMachineConfiguration([]vz.MemoryBalloonDeviceConfiguration{
		memoryBalloonDeviceConfiguration,
	})

	// socket device (optional)
	socketDeviceConfiguration, err := vz.NewVirtioSocketDeviceConfiguration()
	if err != nil {
		return fmt.Errorf("Could not create virtio socket device configuration: %v", err)
	}
	config.SetSocketDevicesVirtualMachineConfiguration([]vz.SocketDeviceConfiguration{
		socketDeviceConfiguration,
	})

	if len(cfg.virtiofsShares) > 0 {
		var cs []vz.DirectorySharingDeviceConfiguration

		for idx, share := range cfg.virtiofsShares {
			tag := fmt.Sprintf("virtiofs%d", idx)
			device, err := vz.NewVirtioFileSystemDeviceConfiguration(tag)
			if err != nil {
				log.Fatal("virtiofs device configuration failed", err)
			}
			dir, err := vz.NewSharedDirectory(share, false)
			if err != nil {
				log.Fatal("virtiofs shared directory failed", err)
			}
			single, err := vz.NewSingleDirectoryShare(dir)
			if err != nil {
				log.Fatal("virtiofs single directory share failed", err)
			}
			device.SetDirectoryShare(single)
			cs = append(cs, device)
		}
		config.SetDirectorySharingDevicesVirtualMachineConfiguration(cs)
	}

	validated, err := config.Validate()
	if !validated || err != nil {
		log.Fatal("validation failed", err)
	}

	vm, err := vz.NewVirtualMachine(config)
	if err != nil {
		return fmt.Errorf("Could not create virtual machine: %v", err)
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGTERM)

	errCh := make(chan error, 1)

	if err := vm.Start(); err != nil {
		errCh <- err
	}

	for {
		select {
		case <-signalCh:
			result, err := vm.RequestStop()
			if err != nil {
				log.Println("request stop error:", err)
				return nil
			}
			log.Println("recieved signal", result)
		case newState := <-vm.StateChangedNotify():
			if newState == vz.VirtualMachineStateRunning {
				log.Println("start VM is running")
			}
			if newState == vz.VirtualMachineStateStopped {
				log.Println("stopped successfully")
				return nil
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
	_ = termios.Tcgetattr(f.Fd(), &attr)

	// Put stdin into raw mode, disabling local echo, input canonicalization,
	// and CR-NL mapping.
	attr.Iflag &^= syscall.ICRNL
	attr.Lflag &^= syscall.ICANON | syscall.ECHO

	// Set minimum characters when reading = 1 char
	attr.Cc[syscall.VMIN] = 1

	// set timeout when reading as non-canonical mode
	attr.Cc[syscall.VTIME] = 0

	// reflects the changed settings
	_ = termios.Tcsetattr(f.Fd(), termios.TCSANOW, &attr)
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
