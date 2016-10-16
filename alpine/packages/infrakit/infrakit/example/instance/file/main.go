package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/docker/infrakit/cli"
	instance_plugin "github.com/docker/infrakit/spi/http/instance"
	"github.com/spf13/cobra"
	"os"
)

func main() {

	var logLevel int
	var name string
	var dir string

	cmd := &cobra.Command{
		Use:   os.Args[0],
		Short: "File instance plugin",
		Run: func(c *cobra.Command, args []string) {

			cli.SetLogLevel(logLevel)
			cli.RunPlugin(name, instance_plugin.PluginServer(NewFileInstancePlugin(dir)))
		},
	}

	cmd.AddCommand(cli.VersionCommand())

	cmd.Flags().StringVar(&name, "name", "instance-file", "Plugin name to advertise for discovery")
	cmd.PersistentFlags().IntVar(&logLevel, "log", cli.DefaultLogLevel, "Logging level. 0 is least verbose. Max is 5")

	cmd.Flags().StringVar(&dir, "dir", os.TempDir(), "Dir for storing the files")

	err := cmd.Execute()
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
}
