package main

import (
	"bytes"
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
	"github.com/spf13/cobra"
)

// VBNetwork is the config for a Virtual Box network
type VBNetwork struct {
	Type    string
	Adapter string
}

// VBNetworks is the type for a list of VBNetwork
type VBNetworks []VBNetwork

func (l *VBNetworks) String() string {
	return fmt.Sprint(*l)
}

func (l *VBNetworks) Type() string {
	return "[]VBNetwork"
}

// Set is used by flag to configure value from CLI
func (l *VBNetworks) Set(value string) error {
	d := VBNetwork{}
	s := strings.Split(value, ",")
	for _, p := range s {
		c := strings.SplitN(p, "=", 2)
		switch len(c) {
		case 1:
			d.Type = c[0]
		case 2:
			switch c[0] {
			case "type":
				d.Type = c[1]
			case "adapter", "bridgeadapter", "hostadapter":
				d.Adapter = c[1]
			default:
				return fmt.Errorf("Unknown network config: %s", c[0])
			}
		}
	}
	*l = append(*l, d)
	return nil
}

func runVBoxCmd() *cobra.Command {
	var (
		enableGUI      bool
		vboxmanageFlag string
		keep           bool
		vmName         string
		state          string
		isoBoot        bool
		uefiBoot       bool
		networks       VBNetworks
	)

	cmd := &cobra.Command{
		Use:   "vbox",
		Short: "launch a vbox VM using an existing image",
		Long: `Launch a vbox VM using an existing image.
		'path' specifies the path to the VM image.
		`,
		Args:    cobra.ExactArgs(1),
		Example: "linuxkit run vbox [options] path",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]
			if runtime.GOOS == "windows" {
				return fmt.Errorf("TODO: Windows is not yet supported")
			}

			if strings.HasSuffix(path, ".iso") {
				isoBoot = true
			}

			vboxmanage, err := exec.LookPath(vboxmanageFlag)
			if err != nil {
				return fmt.Errorf("Cannot find management binary %s: %v", vboxmanageFlag, err)
			}

			name := vmName
			if name == "" {
				name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
			}

			if state == "" {
				prefix := strings.TrimSuffix(path, filepath.Ext(path))
				state = prefix + "-state"
			}
			if err := os.MkdirAll(state, 0755); err != nil {
				return fmt.Errorf("Could not create state directory: %v", err)
			}

			// remove machine in case it already exists
			cleanup(vboxmanage, name, false)

			_, out, err := manage(vboxmanage, "createvm", "--name", name, "--register")
			if err != nil {
				return fmt.Errorf("createvm error: %v\n%s", err, out)
			}

			_, out, err = manage(vboxmanage, "modifyvm", name, "--acpi", "on")
			if err != nil {
				return fmt.Errorf("modifyvm --acpi error: %v\n%s", err, out)
			}

			_, out, err = manage(vboxmanage, "modifyvm", name, "--memory", fmt.Sprintf("%d", mem))
			if err != nil {
				return fmt.Errorf("modifyvm --memory error: %v\n%s", err, out)
			}

			_, out, err = manage(vboxmanage, "modifyvm", name, "--cpus", fmt.Sprintf("%d", cpus))
			if err != nil {
				return fmt.Errorf("modifyvm --cpus error: %v\n%s", err, out)
			}

			firmware := "bios"
			if uefiBoot {
				firmware = "efi"
			}
			_, out, err = manage(vboxmanage, "modifyvm", name, "--firmware", firmware)
			if err != nil {
				return fmt.Errorf("modifyvm --firmware error: %v\n%s", err, out)
			}

			// set up serial console
			_, out, err = manage(vboxmanage, "modifyvm", name, "--uart1", "0x3F8", "4")
			if err != nil {
				return fmt.Errorf("modifyvm --uart error: %v\n%s", err, out)
			}

			var consolePath string
			if runtime.GOOS == "windows" {
				// TODO use a named pipe on Windows
			} else {
				consolePath = filepath.Join(state, "console")
				consolePath, err = filepath.Abs(consolePath)
				if err != nil {
					return fmt.Errorf("Bad path: %v", err)
				}
			}

			_, out, err = manage(vboxmanage, "modifyvm", name, "--uartmode1", "client", consolePath)
			if err != nil {
				return fmt.Errorf("modifyvm --uartmode error: %v\n%s", err, out)
			}

			_, out, err = manage(vboxmanage, "storagectl", name, "--name", "IDE Controller", "--add", "ide")
			if err != nil {
				return fmt.Errorf("storagectl error: %v\n%s", err, out)
			}

			if isoBoot {
				_, out, err = manage(vboxmanage, "storageattach", name, "--storagectl", "IDE Controller", "--port", "1", "--device", "0", "--type", "dvddrive", "--medium", path)
				if err != nil {
					return fmt.Errorf("storageattach error: %v\n%s", err, out)
				}
				_, out, err = manage(vboxmanage, "modifyvm", name, "--boot1", "dvd")
				if err != nil {
					return fmt.Errorf("modifyvm --boot error: %v\n%s", err, out)
				}
			} else {
				_, out, err = manage(vboxmanage, "storageattach", name, "--storagectl", "IDE Controller", "--port", "1", "--device", "0", "--type", "hdd", "--medium", path)
				if err != nil {
					return fmt.Errorf("storageattach error: %v\n%s", err, out)
				}
				_, out, err = manage(vboxmanage, "modifyvm", name, "--boot1", "disk")
				if err != nil {
					return fmt.Errorf("modifyvm --boot error: %v\n%s", err, out)
				}
			}

			if len(disks) > 0 {
				_, out, err = manage(vboxmanage, "storagectl", name, "--name", "SATA", "--add", "sata")
				if err != nil {
					return fmt.Errorf("storagectl error: %v\n%s", err, out)
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
					d.Path = filepath.Join(state, "disk"+id+".img")
					if err := os.Truncate(d.Path, int64(d.Size)*int64(1048576)); err != nil {
						return fmt.Errorf("Cannot create disk: %v", err)
					}
				}
				_, out, err = manage(vboxmanage, "storageattach", name, "--storagectl", "SATA", "--port", "0", "--device", id, "--type", "hdd", "--medium", d.Path)
				if err != nil {
					return fmt.Errorf("storageattach error: %v\n%s", err, out)
				}
			}

			for i, d := range networks {
				nic := i + 1
				_, out, err = manage(vboxmanage, "modifyvm", name, fmt.Sprintf("--nictype%d", nic), "virtio")
				if err != nil {
					return fmt.Errorf("modifyvm --nictype error: %v\n%s", err, out)
				}

				_, out, err = manage(vboxmanage, "modifyvm", name, fmt.Sprintf("--nic%d", nic), d.Type)
				if err != nil {
					return fmt.Errorf("modifyvm --nic error: %v\n%s", err, out)
				}
				if d.Type == "hostonly" {
					_, out, err = manage(vboxmanage, "modifyvm", name, fmt.Sprintf("--hostonlyadapter%d", nic), d.Adapter)
					if err != nil {
						return fmt.Errorf("modifyvm --hostonlyadapter error: %v\n%s", err, out)
					}
				} else if d.Type == "bridged" {
					_, out, err = manage(vboxmanage, "modifyvm", name, fmt.Sprintf("--bridgeadapter%d", nic), d.Adapter)
					if err != nil {
						return fmt.Errorf("modifyvm --bridgeadapter error: %v\n%s", err, out)
					}
				}

				_, out, err = manage(vboxmanage, "modifyvm", name, fmt.Sprintf("--cableconnected%d", nic), "on")
				if err != nil {
					return fmt.Errorf("modifyvm --cableconnected error: %v\n%s", err, out)
				}
			}

			// create socket
			_ = os.Remove(consolePath)
			ln, err := net.Listen("unix", consolePath)
			if err != nil {
				return fmt.Errorf("Cannot listen on console socket %s: %v", consolePath, err)
			}

			var vmType string
			if enableGUI {
				vmType = "gui"
			} else {
				vmType = "headless"
			}

			_, out, err = manage(vboxmanage, "startvm", name, "--type", vmType)
			if err != nil {
				return fmt.Errorf("startvm error: %v\n%s", err, out)
			}

			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt)
			go func() {
				<-c
				cleanup(vboxmanage, name, keep)
				os.Exit(1)
			}()

			socket, err := ln.Accept()
			if err != nil {
				return fmt.Errorf("Accept error: %v", err)
			}

			go func() {
				if _, err := io.Copy(socket, os.Stdin); err != nil {
					cleanup(vboxmanage, name, keep)
					log.Fatalf("Copy error: %v", err)
				}
				cleanup(vboxmanage, name, keep)
				os.Exit(0)
			}()
			go func() {
				if _, err := io.Copy(os.Stdout, socket); err != nil {
					cleanup(vboxmanage, name, keep)
					log.Fatalf("Copy error: %v", err)
				}
				cleanup(vboxmanage, name, keep)
				os.Exit(0)
			}()
			// wait forever
			select {}
		},
	}

	// Display flags
	cmd.Flags().BoolVar(&enableGUI, "gui", false, "Show the VM GUI")

	// vbox options
	cmd.Flags().StringVar(&vboxmanageFlag, "vboxmanage", "VBoxManage", "VBoxManage binary to use")
	cmd.Flags().BoolVar(&keep, "keep", false, "Keep the VM after finishing")
	cmd.Flags().StringVar(&vmName, "name", "", "Name of the Virtualbox VM")
	cmd.Flags().StringVar(&state, "state", "", "Path to directory to keep VM state in")

	// Paths and settings for disks

	// VM configuration

	// booting config
	cmd.Flags().BoolVar(&isoBoot, "iso", false, "Boot image is an ISO")
	cmd.Flags().BoolVar(&uefiBoot, "uefi", false, "Use UEFI boot")

	// networking
	cmd.Flags().Var(&networks, "networking", "Network config, may be repeated. [type=](null|nat|bridged|intnet|hostonly|generic|natnetwork[<devicename>])[,[bridge|host]adapter=<interface>]")

	return cmd
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
