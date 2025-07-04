package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func pkgPushCmd(buildCmd *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push",
		Short: "Alias for 'pkg build --push'",
		Long: `Build and push an OCI package from a directory with a yaml configuration file.
		'path' specifies the path to the package source directory.

		The package may or may not be built first, depending on options
`,
		Example:    `  linuxkit pkg push [options] pkg/dir/`,
		SuggestFor: []string{"build"},
		Args:       cobra.MinimumNArgs(1),
		Deprecated: "use 'pkg build --push' instead",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create a copy of buildCmd with push=true
			if err := buildCmd.Flags().Set("push", "true"); err != nil {
				return fmt.Errorf("'pkg push' unable to set 'pkg build --push': %w", err)
			}

			// Pass the args to the build command
			buildCmd.SetArgs(args)
			return buildCmd.RunE(buildCmd, args)
		},
	}
	cmd.Flags().AddFlagSet(buildCmd.Flags())
	return cmd
}
