package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/registry"
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
		flagQuiet       bool
		flagVerbose     int
		flagVerboseName = "verbose"
		mirrorsRaw      []string
		certFiles       []string
	)
	cmd := &cobra.Command{
		Use:               "linuxkit",
		DisableAutoGenTag: true,
		SilenceUsage:      true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			readConfig()

			// convert the provided mirrors to a map
			for _, m := range mirrorsRaw {
				if m == "" {
					continue
				}
				parts := strings.SplitN(m, "=", 2)
				// if no equals sign, use the whole string as the mirror for all registries
				// not otherwise specified
				var key, value string
				if len(parts) == 1 {
					key = "*"
					value = parts[0]
				} else {
					key = parts[0]
					value = parts[1]
				}
				// value must start with http:// or https://
				if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
					return fmt.Errorf("mirror %q must start with http:// or https://", value)
				}
				// special logic for docker.io because of its odd references
				if key == "docker.io" || key == "docker.io/" {
					for _, prefix := range []string{"docker.io", "index.docker.io", "registry-1.docker.io"} {
						registry.SetProxy(prefix, value)
					}
				} else {
					registry.SetProxy(key, value)
				}
			}

			for _, f := range certFiles {
				if f == "" {
					continue
				}
				cert, err := os.ReadFile(f)
				if err != nil {
					return fmt.Errorf("failed to read certificate file %q: %w", f, err)
				}
				// Add the certificate file to the registry
				registry.AddCert(cert)
			}

			// Set up logging
			return util.SetupLogging(flagQuiet, flagVerbose, cmd.Flag(flagVerboseName).Changed)
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
	cmd.PersistentFlags().StringArrayVar(&mirrorsRaw, "mirror", nil, "Mirror to use for pulling images, format is <registry>=<mirror>, e.g. docker.io=http://mymirror.io, or just http://mymirror.io for all not otherwise specified; must include protocol. Can be provided multiple times.")
	cmd.PersistentFlags().StringArrayVar(&certFiles, "cert-file", nil, "Path to certificate files to use for pulling images, can be provided multiple times. Will augment system-provided certs.")
	cmd.PersistentFlags().BoolVarP(&flagQuiet, "quiet", "q", false, "Quiet execution")
	cmd.PersistentFlags().IntVarP(&flagVerbose, flagVerboseName, "v", 1, "Verbosity of logging: 0 = quiet, 1 = info, 2 = debug, 3 = trace. Default is info. Setting it explicitly will create structured logging lines.")

	return cmd
}
