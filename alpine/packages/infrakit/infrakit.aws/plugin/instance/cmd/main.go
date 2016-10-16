package main

import (
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/infrakit.aws/plugin/instance"
	"github.com/docker/infrakit/cli"
	instance_plugin "github.com/docker/infrakit/spi/http/instance"
	"github.com/spf13/cobra"
)

func main() {

	builder := &instance.Builder{}

	var logLevel int
	var name string
	cmd := &cobra.Command{
		Use:   os.Args[0],
		Short: "AWS instance plugin",
		Run: func(c *cobra.Command, args []string) {

			instancePlugin, err := builder.BuildInstancePlugin()
			if err != nil {
				log.Error(err)
				os.Exit(1)
			}

			cli.SetLogLevel(logLevel)
			cli.RunPlugin(name, instance_plugin.PluginServer(instancePlugin))
		},
	}

	cmd.Flags().IntVar(&logLevel, "log", cli.DefaultLogLevel, "Logging level. 0 is least verbose. Max is 5")
	cmd.Flags().StringVar(&name, "name", "flavor-combo", "Plugin name to advertise for discovery")

	// TODO(chungers) - the exposed flags here won't be set in plugins, because plugin install doesn't allow
	// user to pass in command line args like containers with entrypoint.
	cmd.Flags().AddFlagSet(builder.Flags())

	cmd.AddCommand(cli.VersionCommand())

	err := cmd.Execute()
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
}
