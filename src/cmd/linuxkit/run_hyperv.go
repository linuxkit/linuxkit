package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func runHyperVCmd() *cobra.Command {
	var (
		vmName     string
		keep       bool
		switchName string
	)

	cmd := &cobra.Command{
		Use:   "hyperv",
		Short: "launch a VM in Hyper-V",
		Long: `Launch a VM in Hyper-V.
		'path' specifies the path to a EFI ISO file.
		`,
		Args:    cobra.ExactArgs(1),
		Example: "linuxkit run hyperv [options] path",
		RunE: func(cmd *cobra.Command, args []string) error {
			isoPath := args[0]
			// Sanity checks. Errors out on failure
			hypervChecks()

			vmSwitch, err := hypervGetSwitch(switchName)
			if err != nil {
				return err
			}
			log.Debugf("Using switch: %s", vmSwitch)

			if vmName == "" {
				vmName = filepath.Base(isoPath)
				vmName = strings.TrimSuffix(vmName, ".iso")
				// Also strip -efi in case it is present
				vmName = strings.TrimSuffix(vmName, "-efi")
			}

			log.Infof("Creating VM: %s", vmName)
			_, out, err := poshCmd("New-VM", "-Name", fmt.Sprintf("'%s'", vmName),
				"-Generation", "2",
				"-NoVHD",
				"-SwitchName", fmt.Sprintf("'%s'", vmSwitch))
			if err != nil {
				return fmt.Errorf("Failed to create new VM: %w\n%s", err, out)
			}
			log.Infof("Configure VM: %s", vmName)
			_, out, err = poshCmd("Set-VM", "-Name", fmt.Sprintf("'%s'", vmName),
				"-AutomaticStartAction", "Nothing",
				"-AutomaticStopAction", "ShutDown",
				"-CheckpointType", "Disabled",
				"-MemoryStartupBytes", fmt.Sprintf("%dMB", mem),
				"-StaticMemory",
				"-ProcessorCount", fmt.Sprintf("%d", cpus))
			if err != nil {
				return fmt.Errorf("Failed to configure new VM: %w\n%s", err, out)
			}

			for i, d := range disks {
				id := ""
				if i != 0 {
					id = strconv.Itoa(i)
				}
				if d.Size != 0 && d.Path == "" {
					d.Path = vmName + "-disk" + id + ".vhdx"
				}
				if d.Path == "" {
					return fmt.Errorf("disk specified with no size or name")
				}

				if _, err := os.Stat(d.Path); err != nil {
					if os.IsNotExist(err) {
						log.Infof("Creating new disk %s %dMB", d.Path, d.Size)
						_, out, err = poshCmd("New-VHD",
							"-Path", fmt.Sprintf("'%s'", d.Path),
							"-SizeBytes", fmt.Sprintf("%dMB", d.Size),
							"-Dynamic")
						if err != nil {
							return fmt.Errorf("Failed to create VHD %s: %w\n%s", d.Path, err, out)
						}
					} else {
						return fmt.Errorf("Problem accessing disk %s. %w", d.Path, err)
					}
				} else {
					log.Infof("Using existing disk %s", d.Path)
				}

				_, out, err = poshCmd("Add-VMHardDiskDrive",
					"-VMName", fmt.Sprintf("'%s'", vmName),
					"-Path", fmt.Sprintf("'%s'", d.Path))
				if err != nil {
					return fmt.Errorf("Failed to add VHD %s: %w\n%s", d.Path, err, out)
				}
			}

			log.Info("Setting up boot from ISO")
			_, out, err = poshCmd("Add-VMDvdDrive",
				"-VMName", fmt.Sprintf("'%s'", vmName),
				"-Path", fmt.Sprintf("'%s'", isoPath))
			if err != nil {
				return fmt.Errorf("Failed add DVD: %w\n%s", err, out)
			}
			_, out, err = poshCmd(
				fmt.Sprintf("$cdrom = Get-VMDvdDrive -vmname '%s';", vmName),
				"Set-VMFirmware", "-VMName", fmt.Sprintf("'%s'", vmName),
				"-EnableSecureBoot", "Off",
				"-FirstBootDevice", "$cdrom")
			if err != nil {
				return fmt.Errorf("Failed set DVD as boot device: %w\n%s", err, out)
			}

			log.Info("Set up COM port")
			_, out, err = poshCmd("Set-VMComPort",
				"-VMName", fmt.Sprintf("'%s'", vmName),
				"-number", "1",
				"-Path", fmt.Sprintf(`\\.\pipe\%s-com1`, vmName))
			if err != nil {
				return fmt.Errorf("Failed set up COM port: %w\n%s", err, out)
			}

			log.Info("Start the VM")
			_, out, err = poshCmd("Start-VM", "-Name", fmt.Sprintf("'%s'", vmName))
			if err != nil {
				return fmt.Errorf("Failed start the VM: %w\n%s", err, out)
			}

			err = hypervStartConsole(vmName)
			if err != nil {
				log.Infof("Console returned: %v\n", err)
			}
			hypervRestoreConsole()

			if keep {
				return nil
			}

			log.Info("Stop the VM")
			_, out, err = poshCmd("Stop-VM",
				"-Name", fmt.Sprintf("'%s'", vmName), "-Force")
			if err != nil {
				// Don't error out, could get an error if VM is already stopped
				log.Infof("Stop-VM error: %v\n%s", err, out)
			}

			log.Info("Remove the VM")
			_, out, err = poshCmd("Remove-VM",
				"-Name", fmt.Sprintf("'%s'", vmName), "-Force")
			if err != nil {
				log.Infof("Remove-VM error: %v\n%s", err, out)
			}
			return nil
		},
	}

	//nolint:staticcheck // I honestly have no idea why this is complaining, as this does get called on
	// L159, but anything to get the linter to stop complaining.
	cmd.Flags().BoolVar(&keep, "keep", false, "Keep the VM after finishing")
	cmd.Flags().StringVar(&vmName, "name", "", "Name of the Hyper-V VM")
	cmd.Flags().StringVar(&switchName, "switch", "", "Which Hyper-V switch to attache the VM to. If left empty, either 'Default Switch' or the first external switch found is used.")

	return cmd
}

var powershell string

// Execute a powershell command
func poshCmd(args ...string) (string, string, error) {
	args = append([]string{"-NoProfile", "-NonInteractive"}, args...)
	cmd := exec.Command(powershell, args...)
	log.Debugf("[POSH]: %s %s", powershell, strings.Join(args, " "))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// Perform some sanity checks, and error if failing
func hypervChecks() {
	powershell, _ = exec.LookPath("powershell.exe")
	if powershell == "" {
		log.Fatalf("Could not find powershell executable")
	}

	hvAdmin := false
	admin := false

	out, _, err := poshCmd(`@([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole("Hyper-V Administrators")`)
	if err != nil {
		log.Debugf("Check for Hyper-V Admin failed: %v", err)
	}
	res := splitLines(out)
	if res[0] == "True" {
		hvAdmin = true
	}

	out, _, err = poshCmd(`@([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")`)
	if err != nil {
		log.Debugf("Check for Admin failed: %v", err)
	}
	res = splitLines(out)
	if res[0] == "True" {
		admin = true
	}
	if !hvAdmin && !admin {
		log.Fatal("Must be run from an elevated prompt or user must be in the Hyper-V Administrator role")
	}

	out, _, err = poshCmd("@(Get-Command Get-VM).ModuleName")
	if err != nil {
		log.Fatalf("Check for Hyper-V powershell modules failed: %v", err)
	}
	res = splitLines(out)
	if res[0] != "Hyper-V" {
		log.Fatal("The Hyper-V powershell module does not seem to be installed")
	}
}

// Find a Hyper-V switch. Either check that the supplied switch exists
// or find the first external switch.
func hypervGetSwitch(name string) (string, error) {
	if name != "" {
		if _, _, err := poshCmd("Get-VMSwitch", name); err != nil {
			return "", fmt.Errorf("Could not find switch %s: %v", name, err)
		}
		return name, nil
	}

	// The Windows 10 Fall Creators Update adds a new 'Default
	// Switch'. Check if it is present and if so, use it.
	name = "Default Switch"
	if _, _, err := poshCmd("Get-VMSwitch", name); err == nil {
		return name, nil
	}

	out, _, err := poshCmd("Get-VMSwitch | Format-Table -Property Name, SwitchType -HideTableHeaders")
	if err != nil {
		return "", fmt.Errorf("Could not get list of switches: %v", err)
	}
	switches := splitLines(out)
	for _, s := range switches {
		if len(s) == 0 {
			continue
		}
		t := strings.Split(s, " ")
		if len(t) < 2 {
			continue
		}
		if strings.Contains(t[len(t)-1:][0], "External") {
			return strings.Join(t[:len(t)-1], " "), nil
		}
	}
	return "", fmt.Errorf("Could not find an external switch")
}
