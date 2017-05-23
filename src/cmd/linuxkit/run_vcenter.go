package main

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"

	log "github.com/Sirupsen/logrus"
)

type vmConfig struct {
	vCenterURL  *string
	dsName      *string
	networkName *string
	vSphereHost *string

	vmName       *string
	path         *string
	persistent   *string
	persistentSz int
	vCpus        *int
	mem          *int64
	poweron      *bool
}

func runVcenter(args []string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var newVM vmConfig

	flags := flag.NewFlagSet("vCenter", flag.ExitOnError)
	invoked := filepath.Base(os.Args[0])

	newVM.vCenterURL = flags.String("url", os.Getenv("VCURL"), "URL in the format of https://username:password@host/sdk")
	newVM.dsName = flags.String("datastore", os.Getenv("VMDATASTORE"), "The Name of the DataStore to host the VM")
	newVM.networkName = flags.String("network", os.Getenv("VMNETWORK"), "The VMware vSwitch the VM will use")
	newVM.vSphereHost = flags.String("hostname", os.Getenv("VMHOST"), "The Server that will run the VM")

	newVM.vmName = flags.String("vmname", "", "Specify a name for virtual Machine")
	newVM.path = flags.String("path", "", "Path to a specific image")
	newVM.persistent = flags.String("persistentSize", "", "Size in MB of persistent storage to allocate to the VM")
	newVM.mem = flags.Int64("mem", 1024, "Size in MB of memory to allocate to the VM")
	newVM.vCpus = flags.Int("cpus", 1, "Amount of vCPUs to allocate to the VM")
	newVM.poweron = flags.Bool("powerOn", false, "Power On the new VM once it has been created")

	flags.Usage = func() {
		fmt.Printf("USAGE: %s run vcenter [options] [name]\n\n", invoked)
		fmt.Printf("'name' specifies the full path of an image that will be ran\n")
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

	// Ensure an iso has been passed to the vCenter run Command
	if strings.HasSuffix(*newVM.path, ".iso") {
		// Allow alternative names for new virtual machines being created in vCenter
		if *newVM.vmName == "" {
			*newVM.vmName = strings.TrimSuffix(path.Base(*newVM.path), ".iso")
		}
	} else {
		log.Fatalln("Ensure that an \".iso\" file is used as part of the path")
	}

	// Test any passed in files before creating a new VM
	checkFile(*newVM.path)

	// Parse URL from string
	u, err := url.Parse(*newVM.vCenterURL)
	if err != nil {
		log.Fatalf("URL can't be parsed, ensure it is https://username:password/<address>/sdk")
	}

	// Connect and log in to ESX or vCenter
	c, err := govmomi.NewClient(ctx, u, true)
	if err != nil {
		log.Fatalln("Error logging into vCenter, check address and credentials")
	}

	f := find.NewFinder(c.Client, true)

	// Find one and only datacenter, not sure how VMware linked mode will work
	dc, err := f.DefaultDatacenter(ctx)
	if err != nil {
		log.Fatalln("No Datacenter instance could be found inside of vCenter")
	}

	// Make future calls local to this datacenter
	f.SetDatacenter(dc)

	// Find Datastore/Network
	dss, err := f.DatastoreOrDefault(ctx, *newVM.dsName)
	if err != nil {
		log.Fatalf("Datastore [%s], could not be found", *newVM.dsName)
	}

	net, err := f.NetworkOrDefault(ctx, *newVM.networkName)
	if err != nil {
		log.Fatalf("Network [%s], could not be found", *newVM.networkName)
	}

	// Set the host that the VM will be created on
	hs, err := f.HostSystemOrDefault(ctx, *newVM.vSphereHost)
	if err != nil {
		log.Fatalf("vSphere host [%s], could not be found", *newVM.vSphereHost)
	}

	var rp *object.ResourcePool
	rp, err = hs.ResourcePool(ctx)
	if err != nil {
		log.Fatalln("Error locating default resource pool")
	}

	log.Infof("Creating new LinuxKit Virtual Machine")
	spec := types.VirtualMachineConfigSpec{
		Name:     *newVM.vmName,
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

	folders, err := dc.Folders(ctx)
	if err != nil {
		log.Fatalln("Error locating default datacenter folder")
	}

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

	uploadFile(c, newVM, dss)
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

func uploadFile(c *govmomi.Client, newVM vmConfig, dss *object.Datastore) {
	_, fileName := path.Split(*newVM.path)
	log.Infof("Uploading LinuxKit file [%s]", *newVM.path)
	if *newVM.path == "" {
		log.Fatalf("No file specified")
	}
	dsurl := dss.NewURL(fmt.Sprintf("%s/%s", *newVM.vmName, fileName))

	p := soap.DefaultUpload
	if err := c.Client.UploadFile(*newVM.path, dsurl, &p); err != nil {
		log.Fatalf("Unable to upload file to vCenter Datastore\n%v", err)
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
	disk := devices.CreateDisk(controller, dss.Reference(), dss.Path(fmt.Sprintf("%s/%s", *newVM.vmName, "linuxkit.vmdk")))

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
	add = append(add, devices.InsertIso(cdrom, dss.Path(fmt.Sprintf("%s/%s", *newVM.vmName, path.Base(*newVM.path)))))

	log.Infof("Adding ISO to the Virtual Machine")

	if vm.AddDevice(ctx, add...); err != nil {
		log.Fatalf("Unable to add new CD-ROM device to VM configuration\n%v", err)
	}
}

func checkFile(file string) {
	if _, err := os.Stat(file); err != nil {
		if os.IsPermission(err) {
			log.Fatalf("Unable to read file [%s], please check permissions", file)
		} else if os.IsNotExist(err) {
			log.Fatalf("File [%s], does not exist", file)
		} else {
			log.Fatalf("Unable to stat file [%s]: %v", file, err)
		}
	}
}
