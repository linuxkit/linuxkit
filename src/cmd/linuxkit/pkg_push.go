package main

import "github.com/spf13/cobra"

func pkgPushCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push",
		Short: "build and push an OCI package from a directory with a yaml configuration file",
		Long: `Build and push an OCI package from a directory with a yaml configuration file.
		'path' specifies the path to the package source directory.

		The package may or may not be built first, depending on options
`,
		Example: `  linuxkit pkg push [options] pkg/dir/`,
		Args:    cobra.MinimumNArgs(1),
	}
	return addCmdRunPkgBuildPush(cmd, true)
}
