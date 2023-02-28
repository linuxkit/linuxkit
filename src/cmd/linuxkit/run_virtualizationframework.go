package main

import (
	"github.com/spf13/cobra"
)

const (
	virtualizationNetworkingNone         string = "none"
	virtualizationNetworkingDockerForMac        = "docker-for-mac"
	virtualizationNetworkingVPNKit              = "vpnkit"
	virtualizationNetworkingVMNet               = "vmnet"
	virtualizationNetworkingDefault             = virtualizationNetworkingVMNet
	virtualizationFrameworkConsole              = "console=hvc0"
)

type virtualizationFramwworkConfig struct {
	cpus           uint
	mem            uint64
	disks          Disks
	data           string
	dataPath       string
	state          string
	networking     string
	kernelBoot     bool
	virtiofsShares []string
}

func runVirtualizationFrameworkCmd() *cobra.Command {
	var (
		data           string
		dataPath       string
		state          string
		networking     string
		kernelBoot     bool
		virtiofsShares []string
	)

	cmd := &cobra.Command{
		Use:   "virtualization",
		Short: "launch a VM using the macOS virtualization framework",
		Long: `Launch a VM using the macOS virtualization framework.
		'prefix' specifies the path to the VM image.
		`,
		Args:    cobra.ExactArgs(1),
		Example: "linuxkit run virtualization [options] prefix",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := virtualizationFramwworkConfig{
				cpus:           uint(cpus),
				mem:            uint64(mem) * 1024 * 1024,
				disks:          disks,
				data:           data,
				dataPath:       dataPath,
				state:          state,
				networking:     networking,
				kernelBoot:     kernelBoot,
				virtiofsShares: virtiofsShares,
			}
			return runVirtualizationFramework(cfg, args[0])
		},
	}

	cmd.Flags().StringVar(&data, "data", "", "String of metadata to pass to VM; error to specify both -data and -data-file")
	cmd.Flags().StringVar(&dataPath, "data-file", "", "Path to file containing metadata to pass to VM; error to specify both -data and -data-file")

	cmd.Flags().StringVar(&state, "state", "", "Path to directory to keep VM state in")
	cmd.Flags().StringVar(&networking, "networking", virtualizationNetworkingDefault, "Networking mode. Valid options are 'default', 'vmnet' and 'none'. 'vmnet' uses the Apple vmnet framework. 'none' disables networking.`")

	cmd.Flags().BoolVar(&kernelBoot, "kernel", false, "Boot image is kernel+initrd+cmdline 'path'-kernel/-initrd/-cmdline")
	cmd.Flags().StringArrayVar(&virtiofsShares, "virtiofs", []string{}, "Directory shared on virtiofs")

	return cmd
}
