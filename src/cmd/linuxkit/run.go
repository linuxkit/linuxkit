package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	cpus  int
	mem   int
	disks Disks
)

func runCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "run",
		Short: "run a VM image",
		Long: `Run a VM image.

		'backend' specifies the run backend.
		If the backend is not specified, the platform specific default will be used.
		
		'prefix' specifies the path to the image.
		If the image is not specified, the default is './image'.
		`,
		Example: `run [options] [backend] [prefix]`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var target string
			switch runtime.GOOS {
			case "darwin":
				target = "virtualization"
			case "linux":
				target = "qemu"
			case "windows":
				target = "hyperv"
			default:
				return fmt.Errorf("there currently is no default 'run' backend for your platform.")
			}
			children := cmd.Commands()
			for _, child := range children {
				if child.Name() == target {
					return child.RunE(cmd, args)
				}
			}

			return fmt.Errorf("could not find default for your platform: %s", target)
		},
	}

	// Please keep cases in alphabetical order
	cmd.AddCommand(runAWSCmd())
	cmd.AddCommand(runAzureCmd())
	cmd.AddCommand(runGCPCmd())
	cmd.AddCommand(runHyperkitCmd())
	cmd.AddCommand(runVirtualizationFrameworkCmd())
	cmd.AddCommand(runHyperVCmd())
	cmd.AddCommand(runOpenStackCmd())
	cmd.AddCommand(runPacketCmd())
	cmd.AddCommand(runQEMUCmd())
	cmd.AddCommand(runScalewayCmd())
	cmd.AddCommand(runVMWareCmd())
	cmd.AddCommand(runVBoxCmd())
	cmd.AddCommand(runVCenterCmd())

	cmd.PersistentFlags().IntVar(&cpus, "cpus", 1, "Number of CPUs")
	cmd.PersistentFlags().IntVar(&mem, "mem", 1024, "Amount of memory in MB")
	cmd.PersistentFlags().Var(&disks, "disk", "Disk config. [file=]path[,size=1G]")

	return cmd
}
