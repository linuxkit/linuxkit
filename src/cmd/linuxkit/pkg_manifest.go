package main

import (
	"fmt"

	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/pkglib"
	"github.com/spf13/cobra"
)

func pkgManifestCmd() *cobra.Command {
	var release string
	cmd := &cobra.Command{
		Use:   "manifest",
		Short: "update manifest in the registry for the given path based on all known platforms",
		Long: `Updates the manifest in the registry for the given path based on all known platforms. If none found, no manifest created.
		'path' specifies the path to the package source directory.
`,
		Args:    cobra.ExactArgs(1),
		Aliases: []string{"index"},
		RunE: func(cmd *cobra.Command, args []string) error {
			pkgs, err := pkglib.NewFromConfig(pkglibConfig, args...)
			if err != nil {
				return err
			}

			var opts []pkglib.BuildOpt
			if release != "" {
				opts = append(opts, pkglib.WithRelease(release))
			}

			for _, p := range pkgs {
				msg := fmt.Sprintf("Updating index for %q", p.Tag())
				action := "building and pushing"

				fmt.Println(msg)

				if err := p.Index(opts...); err != nil {
					return fmt.Errorf("error %s %q: %w", action, p.Tag(), err)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&release, "release", "", "Release the given version")

	return cmd
}
