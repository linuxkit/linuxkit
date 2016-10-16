package main

import (
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/infrakit/cli"
	"github.com/docker/infrakit/discovery"
	"github.com/docker/infrakit/plugin/group"
	"github.com/docker/infrakit/spi/flavor"
	flavor_client "github.com/docker/infrakit/spi/http/flavor"
	group_server "github.com/docker/infrakit/spi/http/group"
	instance_client "github.com/docker/infrakit/spi/http/instance"
	"github.com/docker/infrakit/spi/instance"
	"github.com/spf13/cobra"
)

func main() {

	logLevel := cli.DefaultLogLevel
	var name string

	pollInterval := 10 * time.Second

	cmd := &cobra.Command{
		Use:   os.Args[0],
		Short: "Group server",
		RunE: func(c *cobra.Command, args []string) error {

			cli.SetLogLevel(logLevel)

			plugins, err := discovery.NewPluginDiscovery()
			if err != nil {
				return err
			}

			instancePluginLookup := func(n string) (instance.Plugin, error) {
				callable, err := plugins.Find(n)
				if err != nil {
					return nil, err
				}
				return instance_client.PluginClient(callable), nil
			}

			flavorPluginLookup := func(n string) (flavor.Plugin, error) {
				callable, err := plugins.Find(n)
				if err != nil {
					return nil, err
				}
				return flavor_client.PluginClient(callable), nil
			}

			cli.RunPlugin(name, group_server.PluginServer(
				group.NewGroupPlugin(instancePluginLookup, flavorPluginLookup, pollInterval)))

			return nil
		},
	}

	cmd.AddCommand(cli.VersionCommand())

	cmd.Flags().StringVar(&name, "name", "group", "Plugin name to advertise for discovery")
	cmd.Flags().IntVar(&logLevel, "log", logLevel, "Logging level. 0 is least verbose. Max is 5")
	cmd.Flags().DurationVar(&pollInterval, "poll-interval", pollInterval, "Group polling interval")

	err := cmd.Execute()
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
}
