package main

import (
	"fmt"

	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/pkglib"
	"github.com/spf13/cobra"
)

func pkgShowTagCmd() *cobra.Command {
	var canonical bool
	cmd := &cobra.Command{
		Use:   "show-tag",
		Short: "show the tag for packages based on its source directory",
		Long: `Show the tag for one or more packages based on their source directories.
		'path' specifies the path to the package source directory.

When --hash-dir is set, dep tags are read from pre-computed hash files written
by 'linuxkit pkg update-hashes'.  If --build-yml is not explicitly provided and
the default build.yml does not exist, the build-yml field from the hash file is
used instead (e.g. build-2.3.yml for versioned packages like pkg/zfs).

This command does not write hash files; use 'pkg update-hashes' or 'pkg build'
for that.
`,
		Args:    cobra.MinimumNArgs(1),
		Example: "linuxkit pkg show-tag path/to/package [path/to/another/package] [path/to/yet/another/package]",
		RunE: func(cmd *cobra.Command, args []string) error {
			pkgs, err := pkglib.NewFromConfig(pkglibConfig, args...)
			if err != nil {
				return err
			}
			for _, p := range pkgs {
				displayTag := p.Tag()
				if canonical {
					displayTag = p.FullTag()
				}
				fmt.Println(displayTag)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&canonical, "canonical", false, "Show canonical name, e.g. docker.io/linuxkit/foo:1234, instead of the default, e.g. linuxkit/foo:1234")

	return cmd
}
