package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/pkglib"
	"github.com/spf13/cobra"
)

func pkgBuilderCmd() *cobra.Command {
	var (
		builders     string
		platforms    string
		builderImage string
	)
	cmd := &cobra.Command{
		Use:   "builder",
		Short: "manage the pkg builder",
		Long:  `Manage the pkg builder. This normally is a container.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			command := args[0]
			verbose := cmd.Flags().Lookup("verbose").Value.String() == "true"
			// build the builders map
			buildersMap := make(map[string]string)
			// look for builders env var
			buildersMap, err := buildPlatformBuildersMap(os.Getenv(buildersEnvVar), buildersMap)
			if err != nil {
				return fmt.Errorf("invalid environment variable %s: %w", buildersEnvVar, err)
			}
			// any CLI options override env var
			buildersMap, err = buildPlatformBuildersMap(builders, buildersMap)
			if err != nil {
				return fmt.Errorf("invalid --builders flag: %w", err)
			}

			platformsToClean := strings.Split(platforms, ",")
			switch command {
			case "du":
				if err := pkglib.DiskUsage(buildersMap, builderImage, platformsToClean, verbose); err != nil {
					return fmt.Errorf("Unable to print disk usage of builder: %w", err)
				}
			case "prune":
				if err := pkglib.PruneBuilder(buildersMap, builderImage, platformsToClean, verbose); err != nil {
					return fmt.Errorf("Unable to prune builder: %w", err)
				}
			default:
				return fmt.Errorf("unexpected command %s", command)
			}
			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&builders, "builders", "", "Which builders to use for which platforms, e.g. linux/arm64=docker-context-arm64, overrides defaults and environment variables, see https://github.com/linuxkit/linuxkit/blob/master/docs/packages.md#Providing-native-builder-nodes")
	cmd.PersistentFlags().StringVar(&platforms, "platforms", fmt.Sprintf("linux/%s", runtime.GOARCH), "Which platforms we built images for")
	cmd.PersistentFlags().StringVar(&builderImage, "builder-image", defaultBuilderImage, "buildkit builder container image to use")

	return cmd
}
