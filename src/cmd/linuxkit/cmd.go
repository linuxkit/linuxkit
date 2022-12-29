package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/util"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var (
	cacheDir string
	// Config is the global tool configuration
	Config = GlobalConfig{}
)

// GlobalConfig is the global tool configuration
type GlobalConfig struct {
	Pkg PkgConfig `yaml:"pkg"`
}

// PkgConfig is the config specific to the `pkg` subcommand
type PkgConfig struct {
}

func readConfig() {
	cfgPath := filepath.Join(os.Getenv("HOME"), ".moby", "linuxkit", "config.yml")
	cfgBytes, err := os.ReadFile(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		fmt.Printf("Failed to read %q\n", cfgPath)
		os.Exit(1)
	}
	if err := yaml.Unmarshal(cfgBytes, &Config); err != nil {
		fmt.Printf("Failed to parse %q\n", cfgPath)
		os.Exit(1)
	}
}

func newCmd() *cobra.Command {
	var (
		flagQuiet   bool
		flagVerbose bool
	)
	cmd := &cobra.Command{
		Use:               "linuxkit",
		DisableAutoGenTag: true,
		SilenceUsage:      true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			readConfig()

			// Set up logging
			return util.SetupLogging(flagQuiet, flagVerbose)
		},
	}

	cmd.AddCommand(buildCmd()) // apko login
	cmd.AddCommand(cacheCmd())
	cmd.AddCommand(metadataCmd())
	cmd.AddCommand(pkgCmd())
	cmd.AddCommand(pushCmd())
	cmd.AddCommand(runCmd())
	cmd.AddCommand(serveCmd())
	cmd.AddCommand(versionCmd())

	cmd.PersistentFlags().StringVar(&cacheDir, "cache", defaultLinuxkitCache(), fmt.Sprintf("Directory for caching and finding cached image, overrides env var %s", envVarCacheDir))
	cmd.PersistentFlags().BoolVarP(&flagQuiet, "quiet", "q", false, "Quiet execution")
	cmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "Verbose execution")

	return cmd
}
