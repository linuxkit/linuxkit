package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/docker/infrakit/pkg/cli"
	"github.com/docker/infrakit/pkg/plugin/metadata"
	instance_plugin "github.com/docker/infrakit/pkg/rpc/instance"
	metadata_plugin "github.com/docker/infrakit/pkg/rpc/metadata"
	instance_spi "github.com/docker/infrakit/pkg/spi/instance"
)

var (
	// Version is the build release identifier.
	Version = "Unspecified"

	// Revision is the build source control revision.
	Revision = "Unspecified"
)

func main() {

	cmd := &cobra.Command{
		Use:   os.Args[0],
		Short: "HyperKit instance plugin",
	}

	defaultVMDir := filepath.Join(getHome(), ".infrakit/hyperkit-vms")

	name := cmd.Flags().String("name", "instance-hyperkit", "Plugin name to advertise for discovery")
	logLevel := cmd.Flags().Int("log", cli.DefaultLogLevel, "Logging level. 0 is least verbose. Max is 5")

	vmDir := cmd.Flags().String("vm-dir", defaultVMDir, "Directory where to store VM state")
	hyperkit := cmd.Flags().String("hyperkit", "", "Path to HyperKit executable")

	vpnkitSock := cmd.Flags().String("vpnkit-sock", "auto", "Path to VPNKit UNIX domain socket")

	cmd.RunE = func(c *cobra.Command, args []string) error {
		os.MkdirAll(*vmDir, os.ModePerm)

		cli.SetLogLevel(*logLevel)
		cli.RunPlugin(*name,
			instance_plugin.PluginServer(NewHyperKitPlugin(*vmDir, *hyperkit, *vpnkitSock)),
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

func getHome() string {
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return os.Getenv("HOME")
}
