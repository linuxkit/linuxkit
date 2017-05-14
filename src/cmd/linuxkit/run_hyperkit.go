package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
	"github.com/moby/hyperkit/go"
	"github.com/satori/go.uuid"
)

// Process the run arguments and execute run
func runHyperKit(args []string) {
	flags := flag.NewFlagSet("hyperkit", flag.ExitOnError)
	invoked := filepath.Base(os.Args[0])
	flags.Usage = func() {
		fmt.Printf("USAGE: %s run hyperkit [options] prefix\n\n", invoked)
		fmt.Printf("'prefix' specifies the path to the VM image.\n")
		fmt.Printf("\n")
		fmt.Printf("Options:\n")
		flags.PrintDefaults()
	}
	hyperkitPath := flags.String("hyperkit", "", "Path to hyperkit binary (if not in default location)")
	cpus := flags.Int("cpus", 1, "Number of CPUs")
	mem := flags.Int("mem", 1024, "Amount of memory in MB")
	diskSz := flags.Int("disk-size", 0, "Size of Disk in MB")
	disk := flags.String("disk", "", "Path to disk image to used")
	data := flags.String("data", "", "Metadata to pass to VM (either a path to a file or a string)")
	ipStr := flags.String("ip", "", "IP address for the VM")
	state := flags.String("state", "", "Path to directory to keep VM state in")

	if err := flags.Parse(args); err != nil {
		log.Fatal("Unable to parse args")
	}
	remArgs := flags.Args()
	if len(remArgs) == 0 {
		fmt.Println("Please specify the prefix to the image to boot\n")
		flags.Usage()
		os.Exit(1)
	}
	prefix := remArgs[0]

	if *state == "" {
		*state = prefix + "-state"
	}
	if err := os.MkdirAll(*state, 0755); err != nil {
		log.Fatalf("Could not create state directory: %v", err)
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
		isoPath = filepath.Join(*state, "data.iso")
		if err := WriteMetadataISO(isoPath, d); err != nil {
			log.Fatalf("Cannot write user data ISO: %v", err)
		}
	}

	vpnKitKey := ""
	if *ipStr != "" {
		// If an IP address was requested construct a "special" UUID
		// for the VM.
		if ip := net.ParseIP(*ipStr); len(ip) > 0 {
			uuid := make([]byte, 16)
			uuid[12] = ip.To4()[0]
			uuid[13] = ip.To4()[1]
			uuid[14] = ip.To4()[2]
			uuid[15] = ip.To4()[3]
			vpnKitKey = fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:])
		}
	}

	// Generate new UUID, otherwise /sys/class/dmi/id/product_uuid is identical on all VMs
	vmUUID := uuid.NewV4().String()

	// Run
	cmdline, err := ioutil.ReadFile(prefix + "-cmdline")
	if err != nil {
		log.Fatalf("Cannot open cmdline file: %v", err)
	}

	if *diskSz != 0 && *disk == "" {
		*disk = filepath.Join(*state, "disk.img")
	}

	h, err := hyperkit.New(*hyperkitPath, "auto", *state)
	if err != nil {
		log.Fatalln("Error creating hyperkit: ", err)
	}

	h.Kernel = prefix + "-kernel"
	h.Initrd = prefix + "-initrd.img"
	h.VPNKitKey = vpnKitKey
	h.UUID = vmUUID
	h.DiskImage = *disk
	h.ISOImage = isoPath
	h.VSock = true
	h.CPUs = *cpus
	h.Memory = *mem
	h.DiskSize = *diskSz

	err = h.Run(string(cmdline))
	if err != nil {
		log.Fatalf("Cannot run hyperkit: %v", err)
	}
}
