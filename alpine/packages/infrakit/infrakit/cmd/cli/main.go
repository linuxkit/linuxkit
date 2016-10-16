package main

import (
	"errors"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/infrakit/cli"
	"github.com/docker/infrakit/discovery"
	"github.com/spf13/cobra"
)

// A generic client for infrakit

func main() {

	logLevel := cli.DefaultLogLevel

	cmd := &cobra.Command{
		Use:   os.Args[0],
		Short: "infrakit cli",
		PersistentPreRun: func(c *cobra.Command, args []string) {
			cli.SetLogLevel(logLevel)
		},
	}

	cmd.AddCommand(cli.VersionCommand())

	f := func() discovery.Plugins {
		d, err := discovery.NewPluginDiscovery()
		if err != nil {
			log.Fatalf("Failed to initialize plugin discovery: %s", err)
			os.Exit(1)
		}
		return d
	}
	cmd.AddCommand(pluginCommand(f), instancePluginCommand(f), groupPluginCommand(f), flavorPluginCommand(f))

	cmd.PersistentFlags().IntVar(&logLevel, "log", logLevel, "Logging level. 0 is least verbose. Max is 5")

	err := cmd.Execute()
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}

func assertNotNil(message string, f interface{}) {
	if f == nil {
		log.Error(errors.New(message))
		os.Exit(1)
	}
}
