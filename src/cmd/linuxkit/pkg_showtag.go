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
`,
		Args:    cobra.MinimumNArgs(1),
		Example: "linuxkit pkg show-tag path/to/package [path/to/another/package] [path/to/yet/another/package]",
		RunE: func(cmd *cobra.Command, args []string) error {
			pkgs, err := pkglib.NewFromConfig(pkglibConfig, args...)
			if err != nil {
				return err
			}
			for _, p := range pkgs {
				tag := p.Tag()
				if canonical {
					tag = p.FullTag()
				}
				fmt.Println(tag)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&canonical, "canonical", false, "Show canonical name, e.g. docker.io/linuxkit/foo:1234, instead of the default, e.g. linuxkit/foo:1234")

	return cmd
}
