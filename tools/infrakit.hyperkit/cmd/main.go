package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/infrakit/pkg/cli"
	"github.com/docker/infrakit/pkg/plugin/metadata"
	instance_plugin "github.com/docker/infrakit/pkg/rpc/instance"
	metadata_plugin "github.com/docker/infrakit/pkg/rpc/metadata"
	instance_spi "github.com/docker/infrakit/pkg/spi/instance"
	"github.com/spf13/cobra"
)

const (
	// Default path when used with Docker for Mac
	defaultHyperKit = "/Applications/Docker.app/Contents/MacOS/com.docker.hyperkit"
)

var (
	// Version is the build release identifier.
	Version = "Unspecified"

	// Revision is the build source control revision.
	Revision = "Unspecified"

	// Default path to the VPNKit socket on Docker for Mac
	defaultVPNKitSock = "Library/Containers/com.docker.docker/Data/s50"
)

func main() {

	cmd := &cobra.Command{
		Use:   os.Args[0],
		Short: "HyperKit instance plugin",
	}
	defaultVMDir, err := os.Getwd()
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
	defaultVMDir = path.Join(defaultVMDir, "vms")
	homeDir := os.Getenv("HOME")
	defaultVPNKitSock = path.Join(homeDir, defaultVPNKitSock)

	name := cmd.Flags().String("name", "instance-hyperkit", "Plugin name to advertise for discovery")
	logLevel := cmd.Flags().Int("log", cli.DefaultLogLevel, "Logging level. 0 is least verbose. Max is 5")

	vmLib := cmd.Flags().String("vm-lib", "", "Directory with subdirectories of kernels/initrds combinations")
	vmDir := cmd.Flags().String("vm-dir", defaultVMDir, "Directory where to store VM state")
	hyperkit := cmd.Flags().String("hyperkit", defaultHyperKit, "Path to HyperKit executable")

	vpnkitSock := cmd.Flags().String("vpnkit-sock", defaultVPNKitSock, "Path to VPNKit UNIX domain socket")

	cmd.RunE = func(c *cobra.Command, args []string) error {
		cli.SetLogLevel(*logLevel)
		cli.RunPlugin(*name,
			instance_plugin.PluginServer(NewHyperKitPlugin(*vmLib, *vmDir, *hyperkit, *vpnkitSock)),
			metadata_plugin.PluginServer(metadata.NewPluginFromData(
				map[string]interface{}{
					"version":    Version,
					"revision":   Revision,
					"implements": instance_spi.InterfaceSpec,
				},
			)),
		)
		return nil
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "print build version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			buff, err := json.MarshalIndent(map[string]interface{}{
				"version":  Version,
				"revision": Revision,
			}, "  ", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(buff))
			return nil
		},
	})

	if err := cmd.Execute(); err != nil {
		log.Error(err)
		os.Exit(1)
	}
}
