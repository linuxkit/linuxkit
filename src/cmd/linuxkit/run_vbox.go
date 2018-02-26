package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

func runVbox(args []string) {
	invoked := filepath.Base(os.Args[0])
	flags := flag.NewFlagSet("vbox", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Printf("USAGE: %s run vbox [options] path\n\n", invoked)
		fmt.Printf("'path' specifies the path to the VM image.\n")
		fmt.Printf("\n")
		fmt.Printf("Options:\n")
		flags.PrintDefaults()
		fmt.Printf("\n")
	}

	// Display flags
	enableGUI := flags.Bool("gui", false, "Show the VM GUI")

	// vbox options
	vboxmanageFlag := flags.String("vboxmanage", "VBoxManage", "VBoxManage binary to use")
	keep := flags.Bool("keep", false, "Keep the VM after finishing")
	vmName := flags.String("name", "", "Name of the Virtualbox VM")
	state := flags.String("state", "", "Path to directory to keep VM state in")

	// Paths and settings for disks
	var disks Disks
	flags.Var(&disks, "disk", "Disk config, may be repeated. [file=]path[,size=1G][,format=raw]")

	// VM configuration
	cpus := flags.String("cpus", "1", "Number of CPUs")
	mem := flags.String("mem", "1024", "Amount of memory in MB")

	// booting config
	isoBoot := flags.Bool("iso", false, "Boot image is an ISO")
	uefiBoot := flags.Bool("uefi", false, "Use UEFI boot")

	// networking
	networking := flags.String("networking", "nat", "Networking mode. null|nat|bridged|intnet|hostonly|generic|natnetwork[<devicename>]")
	bridgeadapter := flags.String("bridgeadapter", "", "Bridge adapter interface to use if networking mode is bridged")

	if err := flags.Parse(args); err != nil {
		log.Fatal("Unable to parse args")
	}
	remArgs := flags.Args()

	if runtime.GOOS == "windows" {
		log.Fatalf("TODO: Windows is not yet supported")
	}

	if len(remArgs) == 0 {
		fmt.Println("Please specify the path to the image to boot")
		flags.Usage()
		os.Exit(1)
	}
	path := remArgs[0]

	if strings.HasSuffix(path, ".iso") {
		*isoBoot = true
	}

	vboxmanage, err := exec.LookPath(*vboxmanageFlag)
	if err != nil {
		log.Fatalf("Cannot find management binary %s: %v", *vboxmanageFlag, err)
	}

	name := *vmName
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}

	if *state == "" {
		prefix := strings.TrimSuffix(path, filepath.Ext(path))
		*state = prefix + "-state"
	}
	if err := os.MkdirAll(*state, 0755); err != nil {
		log.Fatalf("Could not create state directory: %v", err)
	}

	// remove machine in case it already exists
	cleanup(vboxmanage, name, false)

	_, out, err := manage(vboxmanage, "createvm", "--name", name, "--register")
	if err != nil {
		log.Fatalf("createvm error: %v\n%s", err, out)
	}

	_, out, err = manage(vboxmanage, "modifyvm", name, "--acpi", "on")
	if err != nil {
		log.Fatalf("modifyvm --acpi error: %v\n%s", err, out)
	}

	_, out, err = manage(vboxmanage, "modifyvm", name, "--memory", *mem)
	if err != nil {
		log.Fatalf("modifyvm --memory error: %v\n%s", err, out)
	}

	_, out, err = manage(vboxmanage, "modifyvm", name, "--cpus", *cpus)
	if err != nil {
		log.Fatalf("modifyvm --cpus error: %v\n%s", err, out)
	}

	firmware := "bios"
	if *uefiBoot {
		firmware = "efi"
	}
	_, out, err = manage(vboxmanage, "modifyvm", name, "--firmware", firmware)
	if err != nil {
		log.Fatalf("modifyvm --firmware error: %v\n%s", err, out)
	}

	// set up serial console
	_, out, err = manage(vboxmanage, "modifyvm", name, "--uart1", "0x3F8", "4")
	if err != nil {
		log.Fatalf("modifyvm --uart error: %v\n%s", err, out)
	}

	var consolePath string
	if runtime.GOOS == "windows" {
		// TODO use a named pipe on Windows
	} else {
		consolePath = filepath.Join(*state, "console")
		consolePath, err = filepath.Abs(consolePath)
		if err != nil {
			log.Fatalf("Bad path: %v", err)
		}
	}

	_, out, err = manage(vboxmanage, "modifyvm", name, "--uartmode1", "client", consolePath)
	if err != nil {
		log.Fatalf("modifyvm --uartmode error: %v\n%s", err, out)
	}

	_, out, err = manage(vboxmanage, "storagectl", name, "--name", "IDE Controller", "--add", "ide")
	if err != nil {
		log.Fatalf("storagectl error: %v\n%s", err, out)
	}

	if *isoBoot {
		_, out, err = manage(vboxmanage, "storageattach", name, "--storagectl", "IDE Controller", "--port", "1", "--device", "0", "--type", "dvddrive", "--medium", path)
		if err != nil {
			log.Fatalf("storageattach error: %v\n%s", err, out)
		}
		_, out, err = manage(vboxmanage, "modifyvm", name, "--boot1", "dvd")
		if err != nil {
			log.Fatalf("modifyvm --boot error: %v\n%s", err, out)
		}
	} else {
		_, out, err = manage(vboxmanage, "storageattach", name, "--storagectl", "IDE Controller", "--port", "1", "--device", "0", "--type", "hdd", "--medium", path)
		if err != nil {
			log.Fatalf("storageattach error: %v\n%s", err, out)
		}
		_, out, err = manage(vboxmanage, "modifyvm", name, "--boot1", "disk")
		if err != nil {
			log.Fatalf("modifyvm --boot error: %v\n%s", err, out)
		}
	}

	for i, d := range disks {
		id := strconv.Itoa(i)
		if d.Size != 0 && d.Format == "" {
			d.Format = "raw"
		}
		if d.Format != "raw" && d.Path == "" {
			log.Fatal("vbox currently can only create raw disks")
		}
		if d.Path == "" && d.Size == 0 {
			log.Fatal("please specify an existing disk file or a size")
		}
		if d.Path == "" {
			d.Path = filepath.Join(*state, "disk"+id+".img")
			if err := os.Truncate(d.Path, int64(d.Size)*int64(1048576)); err != nil {
				log.Fatalf("Cannot create disk: %v", err)
			}
		}
		_, out, err = manage(vboxmanage, "storageattach", name, "--storagectl", "IDE Controller", "--port", "2", "--device", id, "--type", "hdd", "--medium", d.Path)
		if err != nil {
			log.Fatalf("storageattach error: %v\n%s", err, out)
		}
	}

	_, out, err = manage(vboxmanage, "modifyvm", name, "--nictype1", "virtio")
	if err != nil {
		log.Fatalf("modifyvm --nictype error: %v\n%s", err, out)
	}

	_, out, err = manage(vboxmanage, "modifyvm", name, "--nic1", *networking)
	if err != nil {
		log.Fatalf("modifyvm --nic error: %v\n%s", err, out)
	}
	if *networking == "bridged" {
		_, out, err = manage(vboxmanage, "modifyvm", name, "--bridgeadapter1", *bridgeadapter)
		if err != nil {
			log.Fatalf("modifyvm --bridgeadapter error: %v\n%s", err, out)
		}
	}

	_, out, err = manage(vboxmanage, "modifyvm", name, "--cableconnected1", "on")
	if err != nil {
		log.Fatalf("modifyvm --cableconnected error: %v\n%s", err, out)
	}

	// create socket
	_ = os.Remove(consolePath)
	ln, err := net.Listen("unix", consolePath)
	if err != nil {
		log.Fatalf("Cannot listen on console socket %s: %v", consolePath, err)
	}

	var vmType string
	if *enableGUI {
		vmType = "gui"
	} else {
		vmType = "headless"
	}

	_, out, err = manage(vboxmanage, "startvm", name, "--type", vmType)
	if err != nil {
		log.Fatalf("startvm error: %v\n%s", err, out)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		cleanup(vboxmanage, name, *keep)
		os.Exit(1)
	}()

	socket, err := ln.Accept()
	if err != nil {
		log.Fatalf("Accept error: %v", err)
	}

	go func() {
		if _, err := io.Copy(socket, os.Stdin); err != nil {
			cleanup(vboxmanage, name, *keep)
			log.Fatalf("Copy error: %v", err)
		}
		cleanup(vboxmanage, name, *keep)
		os.Exit(0)
	}()
	go func() {
		if _, err := io.Copy(os.Stdout, socket); err != nil {
			cleanup(vboxmanage, name, *keep)
			log.Fatalf("Copy error: %v", err)
		}
		cleanup(vboxmanage, name, *keep)
		os.Exit(0)
	}()
	// wait forever
	select {}
}

func cleanup(vboxmanage string, name string, keep bool) {
	_, _, _ = manage(vboxmanage, "controlvm", name, "poweroff")

	if keep {
		return
	}

	// delete VM
	_, _, _ = manage(vboxmanage, "unregistervm", name, "--delete")
}

func manage(vboxmanage string, args ...string) (string, string, error) {
	cmd := exec.Command(vboxmanage, args...)
	log.Debugf("[VBOX]: %s %s", vboxmanage, strings.Join(args, " "))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}
