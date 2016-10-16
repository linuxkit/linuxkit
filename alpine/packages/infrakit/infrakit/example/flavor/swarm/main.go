package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-connections/tlsconfig"
	"github.com/docker/infrakit/cli"
	"github.com/docker/infrakit/plugin/flavor/swarm"
	flavor_plugin "github.com/docker/infrakit/spi/http/flavor"
	"github.com/spf13/cobra"
	"os"
)

func main() {

	var logLevel int
	var name string

	tlsOptions := tlsconfig.Options{}
	host := "unix:///var/run/docker.sock"

	cmd := &cobra.Command{
		Use:   os.Args[0],
		Short: "Docker Swarm flavor plugin",
		Run: func(c *cobra.Command, args []string) {

			cli.SetLogLevel(logLevel)

			dockerClient, err := NewDockerClient(host, &tlsOptions)
			log.Infoln("Connect to docker", host, "err=", err)
			if err != nil {
				log.Error(err)
				os.Exit(1)
			}

			cli.RunPlugin(name, flavor_plugin.PluginServer(swarm.NewSwarmFlavor(dockerClient)))
		},
	}

	cmd.AddCommand(cli.VersionCommand())

	cmd.Flags().StringVar(&name, "name", "flavor-swarm", "Plugin name to advertise for discovery")
	cmd.PersistentFlags().IntVar(&logLevel, "log", cli.DefaultLogLevel, "Logging level. 0 is least verbose. Max is 5")

	cmd.PersistentFlags().StringVar(&host, "host", host, "Docker host")
	cmd.PersistentFlags().StringVar(&tlsOptions.CAFile, "tlscacert", "", "TLS CA cert file path")
	cmd.PersistentFlags().StringVar(&tlsOptions.CertFile, "tlscert", "", "TLS cert file path")
	cmd.PersistentFlags().StringVar(&tlsOptions.KeyFile, "tlskey", "", "TLS key file path")
	cmd.PersistentFlags().BoolVar(&tlsOptions.InsecureSkipVerify, "tlsverify", true, "True to skip TLS")

	err := cmd.Execute()
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
}
