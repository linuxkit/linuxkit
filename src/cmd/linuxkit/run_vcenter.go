package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"

	log "github.com/sirupsen/logrus"
)

type vmConfig struct {
	vCenterURL  *string
	dcName      *string
	dsName      *string
	networkName *string
	vSphereHost *string

	vmFolder     *string
	path         *string
	persistent   *string
	persistentSz int
	vCpus        *int
	mem          *int64
	poweron      *bool
	guestIP      *bool
}

func runVcenter(args []string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var newVM vmConfig

	flags := flag.NewFlagSet("vCenter", flag.ExitOnError)
	invoked := filepath.Base(os.Args[0])

	newVM.vCenterURL = flags.String("url", os.Getenv("VCURL"), "URL of VMware vCenter in the format of https://username:password@VCaddress/sdk")
	newVM.dcName = flags.String("datacenter", os.Getenv("VCDATACENTER"), "The name of the Datacenter to host the VM")
	newVM.dsName = flags.String("datastore", os.Getenv("VCDATASTORE"), "The name of the DataStore to host the VM")
	newVM.networkName = flags.String("network", os.Getenv("VCNETWORK"), "The network label the VM will use")
	newVM.vSphereHost = flags.String("hostname", os.Getenv("VCHOST"), "The server that will run the VM")

	newVM.vmFolder = flags.String("vmfolder", "", "Specify a name/folder for the virtual machine to reside in")
	newVM.path = flags.String("path", "", "Path to a specific image")
	newVM.persistent = flags.String("persistentSize", "", "Size in MB of persistent storage to allocate to the VM")
	newVM.mem = flags.Int64("mem", 1024, "Size in MB of memory to allocate to the VM")
	newVM.vCpus = flags.Int("cpus", 1, "Amount of vCPUs to allocate to the VM")
	newVM.poweron = flags.Bool("powerOn", false, "Power On the new VM once it has been created")
	newVM.guestIP = flags.Bool("waitForIP", false, "LinuxKit will wait for the VM to power on and return the guest IP, requires open-vm-tools and the -powerOn flag to be set")

	flags.Usage = func() {
		fmt.Printf("USAGE: %s run vcenter [options] path\n\n", invoked)
		fmt.Printf("'path' specifies the full path of an image to run\n")
		fmt.Printf("Options:\n\n")
		flags.PrintDefaults()
	}

	if err := flags.Parse(args); err != nil {
		log.Fatalln("Unable to parse args")
	}

	remArgs := flags.Args()
	if len(remArgs) == 0 {
		fmt.Printf("Please specify the path to the image to run\n")
		flags.Usage()
		os.Exit(1)
	}
	*newVM.path = remArgs[0]

	if (*newVM.guestIP == true) && *newVM.poweron != true {
		log.Fatalln("The waitForIP flag can not be used without the powerOn flag")
	}
	// Ensure an iso has been passed to the vCenter run Command
	if strings.HasSuffix(*newVM.path, ".iso") {
		// Allow alternative names for new virtual machines being created in vCenter
		if *newVM.vmFolder == "" {
			*newVM.vmFolder = strings.TrimSuffix(path.Base(*newVM.path), ".iso")
		}
	} else {
		log.Fatalln("Please pass an \".iso\" file as the path")
	}

	// Connect to VMware vCenter and return the default and found values needed for a new VM
	c, dss, folders, hs, net, rp := vCenterConnect(ctx, newVM)

	log.Infof("Creating new LinuxKit Virtual Machine")
	spec := types.VirtualMachineConfigSpec{
		Name:     *newVM.vmFolder,
		GuestId:  "otherLinux64Guest",
		Files:    &types.VirtualMachineFileInfo{VmPathName: fmt.Sprintf("[%s]", dss.Name())},
		NumCPUs:  int32(*newVM.vCpus),
		MemoryMB: *newVM.mem,
	}

	scsi, err := object.SCSIControllerTypes().CreateSCSIController("pvscsi")
	if err != nil {
		log.Fatalln("Error creating pvscsi controller as part of new VM")
	}

	spec.DeviceChange = append(spec.DeviceChange, &types.VirtualDeviceConfigSpec{
		Operation: types.VirtualDeviceConfigSpecOperationAdd,
		Device:    scsi,
	})

	task, err := folders.VmFolder.CreateVM(ctx, spec, rp, hs)
	if err != nil {
		log.Fatalln("Creating new VM failed, more detail can be found in vCenter tasks")
	}

	info, err := task.WaitForResult(ctx, nil)
	if err != nil {
		log.Fatalf("Creating new VM failed\n%v", err)
	}

	// Retrieve the new VM
	vm := object.NewVirtualMachine(c.Client, info.Result.(types.ManagedObjectReference))

	addISO(ctx, newVM, vm, dss)

	if *newVM.persistent != "" {
		newVM.persistentSz, err = getDiskSizeMB(*newVM.persistent)
		if err != nil {
			log.Fatalf("Couldn't parse disk-size %s: %v", *newVM.persistent, err)
		}
		addVMDK(ctx, vm, dss, newVM)
	}

	if *newVM.networkName != "" {
		addNIC(ctx, vm, net)
	}

	if *newVM.poweron == true {
		log.Infoln("Powering on LinuxKit VM")
		powerOnVM(ctx, vm)
	}

	if *newVM.guestIP {
		log.Infof("Waiting for OpenVM Tools to come online")
		guestIP, err := getVMToolsIP(ctx, vm)
		if err != nil {
			log.Errorf("%v", err)
		}
		log.Infof("Guest IP Address: %s", guestIP)
	}
}

func getVMToolsIP(ctx context.Context, vm *object.VirtualMachine) (string, error) {
	guestIP, err := vm.WaitForIP(ctx)
	if err != nil {
		return "", err
	}
	return guestIP, err
}

func vCenterConnect(ctx context.Context, newVM vmConfig) (*govmomi.Client, *object.Datastore, *object.DatacenterFolders, *object.HostSystem, object.NetworkReference, *object.ResourcePool) {

	// Parse URL from string
	u, err := url.Parse(*newVM.vCenterURL)
	if err != nil {
		log.Fatalf("URL can't be parsed, ensure it is https://username:password/<address>/sdk %v", err)
	}

	// Connect and log in to ESX or vCenter
	c, err := govmomi.NewClient(ctx, u, true)
	if err != nil {
		log.Fatalf("Error logging into vCenter, check address and credentials %v", err)
	}

	// Create a new finder that will discover the defaults and are looked for Networks/Datastores
	f := find.NewFinder(c.Client, true)

	// Find one and only datacenter, not sure how VMware linked mode will work
	dc, err := f.DatacenterOrDefault(ctx, *newVM.dcName)
	if err != nil {
		log.Fatalf("No Datacenter instance could be found inside of vCenter %v", err)
	}

	// Make future calls local to this datacenter
	f.SetDatacenter(dc)

	// Find Datastore/Network
	dss, err := f.DatastoreOrDefault(ctx, *newVM.dsName)
	if err != nil {
		log.Fatalf("Datastore [%s], could not be found", *newVM.dsName)
	}

	folders, err := dc.Folders(ctx)
	if err != nil {
		log.Fatalln("Error locating default datacenter folder")
	}

	// This code is shared between Push/Run, a network isn't needed for Pushing an image
	var net object.NetworkReference
	if newVM.networkName != nil && *newVM.networkName != "" {
		net, err = f.NetworkOrDefault(ctx, *newVM.networkName)
		if err != nil {
			log.Fatalf("Network [%s], could not be found", *newVM.networkName)
		}
	}

	// Check to see if a host has been specified
	var hs *object.HostSystem
	if *newVM.vSphereHost != "" {
		// Find the selected host
		hs, err = f.HostSystem(ctx, *newVM.vSphereHost)
		if err != nil {
			log.Fatalf("vSphere host [%s], could not be found", *newVM.vSphereHost)
		}
	} else {
		allHosts, err := f.HostSystemList(ctx, "*/*")
		if err != nil || len(allHosts) == 0 {
			log.Fatalf("No vSphere hosts could be found on vCenter server")
		}
		// Select a host from the list off all available hosts at random
		hs = allHosts[rand.Int()%len(allHosts)]
	}

	var rp *object.ResourcePool
	rp, err = hs.ResourcePool(ctx)
	if err != nil {
		log.Fatalln("Error locating default resource pool")
	}
	return c, dss, folders, hs, net, rp
}

func powerOnVM(ctx context.Context, vm *object.VirtualMachine) {
	task, err := vm.PowerOn(ctx)
	if err != nil {
		log.Errorln("Power On operation has failed, more detail can be found in vCenter tasks")
	}

	_, err = task.WaitForResult(ctx, nil)
	if err != nil {
		log.Errorln("Power On Task has failed, more detail can be found in vCenter tasks")
	}
}

func addNIC(ctx context.Context, vm *object.VirtualMachine, net object.NetworkReference) {
	backing, err := net.EthernetCardBackingInfo(ctx)
	if err != nil {
		log.Fatalf("Unable to determine vCenter network backend\n%v", err)
	}

	netdev, err := object.EthernetCardTypes().CreateEthernetCard("vmxnet3", backing)
	if err != nil {
		log.Fatalf("Unable to create vmxnet3 network interface\n%v", err)
	}

	log.Infof("Adding VM Networking")
	var add []types.BaseVirtualDevice
	add = append(add, netdev)

	if vm.AddDevice(ctx, add...); err != nil {
		log.Fatalf("Unable to add new networking device to VM configuration\n%v", err)
	}
}

func addVMDK(ctx context.Context, vm *object.VirtualMachine, dss *object.Datastore, newVM vmConfig) {
	devices, err := vm.Device(ctx)
	if err != nil {
		log.Fatalf("Unable to read devices from VM configuration\n%v", err)
	}

	controller, err := devices.FindDiskController("scsi")
	if err != nil {
		log.Fatalf("Unable to find SCSI device from VM configuration\n%v", err)
	}
	// The default is to have all persistent disks named linuxkit.vmdk
	disk := devices.CreateDisk(controller, dss.Reference(), dss.Path(fmt.Sprintf("%s/%s", *newVM.vmFolder, "linuxkit.vmdk")))

	disk.CapacityInKB = int64(newVM.persistentSz * 1024)

	var add []types.BaseVirtualDevice
	add = append(add, disk)

	log.Infof("Adding a persistent disk to the Virtual Machine")

	if vm.AddDevice(ctx, add...); err != nil {
		log.Fatalf("Unable to add new storage device to VM configuration\n%v", err)
	}
}

func addISO(ctx context.Context, newVM vmConfig, vm *object.VirtualMachine, dss *object.Datastore) {
	devices, err := vm.Device(ctx)
	if err != nil {
		log.Fatalf("Unable to read devices from VM configuration\n%v", err)
	}

	ide, err := devices.FindIDEController("")
	if err != nil {
		log.Fatalf("Unable to find IDE device from VM configuration\n%v", err)
	}

	cdrom, err := devices.CreateCdrom(ide)
	if err != nil {
		log.Fatalf("Unable to create new CDROM device\n%v", err)
	}

	var add []types.BaseVirtualDevice
	add = append(add, devices.InsertIso(cdrom, dss.Path(fmt.Sprintf("%s/%s", *newVM.vmFolder, path.Base(*newVM.path)))))

	log.Infof("Adding ISO to the Virtual Machine")

	if vm.AddDevice(ctx, add...); err != nil {
		log.Fatalf("Unable to add new CD-ROM device to VM configuration\n%v", err)
	}
}
