//go:build darwin && cgo
// +build darwin,cgo

package main

import (
	"context"
	"errors"
	"fmt"
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
		return errors.New("cannot specify both -data and -data-file")
	}

	prefix := path

	_, err := os.Stat(path + "-kernel")
	statKernel := err == nil

	var isoPaths []string

	// Default to kernel+initrd
	if !statKernel {
		return fmt.Errorf("cannot find kernel file: %s", path+"-kernel")
	}
	_, err = os.Stat(path + "-initrd.img")
	statInitrd := err == nil
	if !statInitrd {
		return fmt.Errorf("cannot find initrd file (%s): %w", path+"-initrd.img", err)
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
		return fmt.Errorf("cannot open cmdline file: %v", err)
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

	// Handle signals during kernel preparation and VM lifecycle; earlier metadata/cmdline reads keep default termination.
	ctx, forceStop, cleanupSignals := virtualizationFrameworkSignalContext()
	defer cleanupSignals()

	vmlinuzFile, cleanupKernel, err := prepareVirtualizationFrameworkKernelContext(ctx, vmlinuz)
	if err != nil {
		return err
	}
	defer cleanupKernel()

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
	restoreTerminal := setRawMode(os.Stdin)
	defer restoreTerminal()

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
			return fmt.Errorf("could not create NAT network device attachment: %v", err)
		}
		networkConfig, err := vz.NewVirtioNetworkDeviceConfiguration(natAttachment)
		if err != nil {
			return fmt.Errorf("could not create virtio network device configuration: %v", err)
		}
		config.SetNetworkDevicesVirtualMachineConfiguration([]*vz.VirtioNetworkDeviceConfiguration{
			networkConfig,
		})
		macAddress, err := vz.NewRandomLocallyAdministeredMACAddress()
		if err != nil {
			return fmt.Errorf("could not create random MAC address: %v", err)
		}
		networkConfig.SetMACAddress(macAddress)
	case virtualizationNetworkingNone:
	default:
		return fmt.Errorf("invalid networking mode: %s", netMode[0])
	}

	// entropy
	entropyConfig, err := vz.NewVirtioEntropyDeviceConfiguration()
	if err != nil {
		return fmt.Errorf("could not create virtio entropy device configuration: %v", err)
	}

	config.SetEntropyDevicesVirtualMachineConfiguration([]*vz.VirtioEntropyDeviceConfiguration{
		entropyConfig,
	})

	var storageDevices []vz.StorageDeviceConfiguration
	for i, d := range cfg.disks {
		var id string
		if i != 0 {
			id = strconv.Itoa(i)
		}
		if d.Size != 0 && d.Format == "" {
			d.Format = "raw"
		}
		if d.Format != "raw" && d.Path == "" {
			return fmt.Errorf("Virtualization Framework currently can only create raw disks")
		}
		if d.Size != 0 && d.Path == "" {
			d.Path = filepath.Join(cfg.state, "disk"+id+".raw")
		}
		if d.Path == "" {
			return fmt.Errorf("disk specified with no size or name")
		}
		if _, err := os.Stat(d.Path); err != nil {
			if !os.IsNotExist(err) {
				return err
			}
			err = vz.CreateDiskImage(d.Path, int64(d.Size)*1024*1024)
			if err != nil {
				return err
			}
		}
		diskImageAttachment, err := vz.NewDiskImageStorageDeviceAttachment(
			d.Path,
			false,
		)
		if err != nil {
			return fmt.Errorf("could not attach storage image: %w", err)
		}
		storageDeviceConfig, err := vz.NewVirtioBlockDeviceConfiguration(diskImageAttachment)
		if err != nil {
			return fmt.Errorf("could not create virtio block device configuration: %v", err)
		}
		storageDevices = append(storageDevices, storageDeviceConfig)
	}
	for _, iso := range isoPaths {
		diskImageAttachment, err := vz.NewDiskImageStorageDeviceAttachment(
			iso,
			true,
		)
		if err != nil {
			return fmt.Errorf("could not attach storage image: %w", err)
		}
		storageDeviceConfig, err := vz.NewVirtioBlockDeviceConfiguration(diskImageAttachment)
		if err != nil {
			return fmt.Errorf("could not create virtio block device configuration: %v", err)
		}
		storageDevices = append(storageDevices, storageDeviceConfig)
	}

	config.SetStorageDevicesVirtualMachineConfiguration(storageDevices)

	// traditional memory balloon device which allows for managing guest memory. (optional)
	memoryBalloonDeviceConfiguration, err := vz.NewVirtioTraditionalMemoryBalloonDeviceConfiguration()
	if err != nil {
		return fmt.Errorf("could not create virtio traditional memory balloon device configuration: %v", err)
	}
	config.SetMemoryBalloonDevicesVirtualMachineConfiguration([]vz.MemoryBalloonDeviceConfiguration{
		memoryBalloonDeviceConfiguration,
	})

	// socket device (optional)
	socketDeviceConfiguration, err := vz.NewVirtioSocketDeviceConfiguration()
	if err != nil {
		return fmt.Errorf("could not create virtio socket device configuration: %v", err)
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
				return fmt.Errorf("virtiofs device configuration failed: %w", err)
			}
			dir, err := vz.NewSharedDirectory(share, false)
			if err != nil {
				return fmt.Errorf("virtiofs shared directory failed: %w", err)
			}
			single, err := vz.NewSingleDirectoryShare(dir)
			if err != nil {
				return fmt.Errorf("virtiofs single directory share failed: %w", err)
			}
			device.SetDirectoryShare(single)
			cs = append(cs, device)
		}
		config.SetDirectorySharingDevicesVirtualMachineConfiguration(cs)
	}

	validated, err := config.Validate()
	if err != nil {
		return fmt.Errorf("VM configuration validation failed: %w", err)
	}
	if !validated {
		return fmt.Errorf("VM configuration is invalid")
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}

	vm, err := vz.NewVirtualMachine(config)
	if err != nil {
		return fmt.Errorf("could not create virtual machine: %v", err)
	}

	timeouts := defaultVirtualizationFrameworkLifecycleTimeouts()
	err = runVirtualizationFrameworkVMLifecycle(
		ctx,
		forceStop,
		virtualizationFrameworkVMAdapter{vm: vm},
		vm.StateChangedNotify(),
		vz.VirtualMachineStateStopped,
		vz.VirtualMachineStateRunning,
		vz.VirtualMachineStateError,
		timeouts,
		virtualizationFrameworkLifecycleHooks{
			Logf:      log.Infof,
			OnRunning: func() { log.Infof("start VM is running") },
		},
	)
	return err

}

type virtualizationFrameworkVMAdapter struct {
	vm *vz.VirtualMachine
}

func (v virtualizationFrameworkVMAdapter) Start() error {
	return v.vm.Start()
}

func (v virtualizationFrameworkVMAdapter) RequestStop() (bool, error) {
	return v.vm.RequestStop()
}

func (v virtualizationFrameworkVMAdapter) Stop() error {
	return v.vm.Stop()
}

func virtualizationFrameworkSignalContext() (context.Context, <-chan struct{}, func()) {
	signalCh := make(chan os.Signal, 2)
	signal.Notify(signalCh, syscall.SIGTERM, os.Interrupt)
	ctx, cancel := context.WithCancel(context.Background())
	forceStop := make(chan struct{})
	stopWatcher := make(chan struct{})
	watcherDone := make(chan struct{})

	go func() {
		defer close(watcherDone)
		firstSignal := true
		for {
			select {
			case sig := <-signalCh:
				if firstSignal {
					firstSignal = false
					log.Infof("received signal %s; requesting graceful virtual machine shutdown", sig)
					cancel()
					continue
				}
				log.Warnf("received signal %s again; forcing virtual machine shutdown", sig)
				close(forceStop)
				return
			case <-stopWatcher:
				return
			}
		}
	}()

	cleanup := func() {
		signal.Stop(signalCh)
		close(stopWatcher)
		cancel()
		<-watcherDone
	}
	return ctx, forceStop, cleanup
}

// https://developer.apple.com/documentation/virtualization/running_linux_in_a_virtual_machine?language=objc#:~:text=Configure%20the%20Serial%20Port%20Device%20for%20Standard%20In%20and%20Out
func setRawMode(f *os.File) func() {
	var attr unix.Termios

	// Get settings for terminal
	if err := termios.Tcgetattr(f.Fd(), &attr); err != nil {
		return func() {}
	}
	original := attr

	// Put stdin into raw mode, disabling local echo, input canonicalization,
	// and CR-NL mapping.
	attr.Iflag &^= syscall.ICRNL
	attr.Lflag &^= syscall.ICANON | syscall.ECHO

	// Set minimum characters when reading = 1 char
	attr.Cc[syscall.VMIN] = 1

	// set timeout when reading as non-canonical mode
	attr.Cc[syscall.VTIME] = 0

	// reflects the changed settings
	if err := termios.Tcsetattr(f.Fd(), termios.TCSANOW, &attr); err != nil {
		return func() {}
	}
	return func() { _ = termios.Tcsetattr(f.Fd(), termios.TCSANOW, &original) }
}
