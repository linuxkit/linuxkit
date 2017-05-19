package main

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"

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

	vmName     *string
	iso        *bool
	disk       *bool
	isoPath    *string
	diskPath   *string
	persistent *int64
	vCpus      *int
	mem        *int64
}

func exit(err error) {
	log.Fatalf("Error: %s\n", err)
	os.Exit(1)
}

func pushVcenter(args []string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var newVM vmConfig

	flags := flag.NewFlagSet("vCenter", flag.ExitOnError)
	invoked := filepath.Base(os.Args[0])

	newVM.vCenterURL = flags.String("url", os.Getenv("STEVEDOR_URL"), "https://username:password@host/sdk")
	newVM.dsName = flags.String("datastore", "", "The Name of the DataStore to host the VM")
	newVM.networkName = flags.String("network", os.Getenv("VMNETWORK"), "The VMware vSwitch the VM will use")
	newVM.vSphereHost = flags.String("hostname", os.Getenv("VMHOST"), "The Server that will run the VM")

	newVM.vmName = flags.String("vmname", "", "Specify a name for virtual Machine")
	newVM.iso = flags.Bool("iso", false, "Push LinuxKit ISO image to vCenter")
	newVM.disk = flags.Bool("disk", false, "Push VMware VMDK to vCenter")
	newVM.isoPath = flags.String("isopath", "", "Specify the path to the VM ISO")
	newVM.diskPath = flags.String("diskpath", "", "Specify the path to the VMware VMDK file")
	newVM.persistent = flags.Int64("persistentSize", 0, "Size in MB of persistent storage to allocate to the VM")
	newVM.mem = flags.Int64("mem", 1024, "Size in MB of memory to allocate to the VM")
	newVM.vCpus = flags.Int("cpus", 1, "Amount of vCPUs to allocate to the VM")

	flags.Usage = func() {
		fmt.Printf("USAGE: %s push vcenter [options] [name]\n\n", invoked)
		fmt.Printf("'name' specifies the full path of an image file which will be uploaded\n")
		fmt.Printf("Options:\n\n")
		flags.PrintDefaults()
	}

	if err := flags.Parse(args); err != nil {
		log.Fatal("Unable to parse args")
	}

	remArgs := flags.Args()
	if len(remArgs) == 0 {
		fmt.Printf("Please specify the prefix to the image to push\n")
		flags.Usage()
		os.Exit(1)
	}
	prefix := remArgs[0]

	if *newVM.vmName == "" {
		*newVM.vmName = prefix
	}

	// Parse URL from string
	u, err := url.Parse(*newVM.vCenterURL)
	if err != nil {
		exit(err)
	}

	// test any passed in files before creating a new VM
	if *newVM.iso == true && *newVM.isoPath == "" {
		*newVM.isoPath = prefix + ".iso"
	}
	if *newVM.disk == true && *newVM.diskPath == "" {
		*newVM.diskPath = prefix + ".vmdk"
	}

	if *newVM.isoPath != "" {
		checkFile(*newVM.isoPath)
	}
	if *newVM.diskPath != "" {
		checkFile(*newVM.diskPath)
	}

	// Connect and log in to ESX or vCenter
	c, err := govmomi.NewClient(ctx, u, true)
	if err != nil {
		exit(err)
	}

	f := find.NewFinder(c.Client, true)

	// Find one and only datacenter, not sure how VMware linked mode will work
	dc, err := f.DefaultDatacenter(ctx)
	if err != nil {
		exit(err)
	}

	// Make future calls local to this datacenter
	f.SetDatacenter(dc)

	dss, err := f.DatastoreOrDefault(ctx, *newVM.dsName)
	if err != nil {
		exit(err)
	}

	net, err := f.NetworkOrDefault(ctx, *newVM.networkName)
	if err != nil {
		exit(err)
	}

	hs, err := f.HostSystemOrDefault(ctx, *newVM.vSphereHost)
	if err != nil {
		exit(err)
	}

	var rp *object.ResourcePool
	rp, err = hs.ResourcePool(ctx)
	if err != nil {
		exit(err)
	}

	spec := types.VirtualMachineConfigSpec{
		Name:     *newVM.vmName,
		GuestId:  "otherLinux64Guest",
		Files:    &types.VirtualMachineFileInfo{VmPathName: fmt.Sprintf("[%s]", dss.Name())},
		NumCPUs:  int32(*newVM.vCpus),
		MemoryMB: *newVM.mem,
	}

	scsi, err := object.SCSIControllerTypes().CreateSCSIController("pvscsi")
	if err != nil {
		exit(err)
	}

	spec.DeviceChange = append(spec.DeviceChange, &types.VirtualDeviceConfigSpec{
		Operation: types.VirtualDeviceConfigSpecOperationAdd,
		Device:    scsi,
	})

	log.Infof("Creating initial LinuxKit Virtual Machine")
	folders, err := dc.Folders(ctx)
	if err != nil {
		exit(err)
	}

	task, err := folders.VmFolder.CreateVM(ctx, spec, rp, hs)
	if err != nil {
		exit(err)
	}

	info, err := task.WaitForResult(ctx, nil)
	if err != nil {
		exit(err)
	}

	// Retrieve the new VM
	vm := object.NewVirtualMachine(c.Client, info.Result.(types.ManagedObjectReference))

	if *newVM.isoPath != "" {
		uploadFile(c, newVM, dss)
		addISO(ctx, newVM, vm, dss)
	}

	if *newVM.diskPath != "" {
		uploadFile(c, newVM, dss)
		addVMDK(ctx, vm, dss, newVM)
	}

	if *newVM.persistent != 0 {
		if *newVM.diskPath != "linuxkit.vmdk" {
			addVMDK(ctx, vm, dss, newVM)
		} else {
			log.Errorf("Can not create persisten disk with identical name to existing VMDK disk")
		}
	}

	if *newVM.networkName != "" {
		addNIC(ctx, vm, net)
	}

}

func uploadFile(c *govmomi.Client, newVM vmConfig, dss *object.Datastore) {
	_, fileName := path.Split(*newVM.isoPath)
	log.Infof("Uploading LinuxKit file [%s]", *newVM.isoPath)
	if *newVM.isoPath == "" {
		log.Fatalf("No file specified")
	}
	dsurl := dss.NewURL(fmt.Sprintf("%s/%s", *newVM.vmName, fileName))

	p := soap.DefaultUpload
	if err := c.Client.UploadFile(*newVM.isoPath, dsurl, &p); err != nil {
		exit(err)
	}
}

func addNIC(ctx context.Context, vm *object.VirtualMachine, net object.NetworkReference) {
	backing, err := net.EthernetCardBackingInfo(ctx)
	if err != nil {
		exit(err)
	}

	netdev, err := object.EthernetCardTypes().CreateEthernetCard("vmxnet3", backing)
	if err != nil {
		exit(err)

	}

	log.Infof("Adding VM Networking")
	var add []types.BaseVirtualDevice
	add = append(add, netdev)

	if vm.AddDevice(ctx, add...); err != nil {
		exit(err)
	}
}

func addVMDK(ctx context.Context, vm *object.VirtualMachine, dss *object.Datastore, newVM vmConfig) {
	devices, err := vm.Device(ctx)
	if err != nil {
		exit(err)
	}

	controller, err := devices.FindDiskController("scsi")
	if err != nil {
		exit(err)
	}

	_, vmdkName := path.Split(*newVM.diskPath)
	disk := devices.CreateDisk(controller, dss.Reference(), dss.Path(fmt.Sprintf("%s/%s", *newVM.vmName, vmdkName)))

	disk.CapacityInKB = *newVM.persistent * 1024

	var add []types.BaseVirtualDevice
	add = append(add, disk)

	log.Infof("Adding the new disk to the Virtual Machine")

	if vm.AddDevice(ctx, add...); err != nil {
		exit(err)
	}
}

func addISO(ctx context.Context, newVM vmConfig, vm *object.VirtualMachine, dss *object.Datastore) {
	devices, err := vm.Device(ctx)
	if err != nil {
		exit(err)
	}

	ide, err := devices.FindIDEController("")
	if err != nil {
		exit(err)
	}

	cdrom, err := devices.CreateCdrom(ide)
	if err != nil {
		exit(err)
	}

	var add []types.BaseVirtualDevice
	add = append(add, devices.InsertIso(cdrom, dss.Path(fmt.Sprintf("%s/%s", *newVM.vmName, "linuxkit.iso"))))

	log.Infof("Adding ISO to the Virtual Machine")

	if vm.AddDevice(ctx, add...); err != nil {
		exit(err)
	}
}

func checkFile(file string) {
	if _, err := os.Stat(file); err != nil {
		if os.IsPermission(err) {
			log.Fatalf("Unable to read file [%s], please check permissions", file)
		}
		if os.IsNotExist(err) {
			log.Fatalf("File [%s], does not exist", file)
		}
	}
}
