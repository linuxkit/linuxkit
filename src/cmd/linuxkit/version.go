package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/version"
	"github.com/spf13/cobra"
)

func versionCmd() *cobra.Command {
	var short, commit bool
	cmd := &cobra.Command{
		Use:   "version",
		Short: "report the version of linuxkit",
		Long: `Report the version of linuxkit.
		Run with option --short to print just the version number.
		Run with option --commit to print just the git commit.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if short {
				fmt.Println(version.Version)
				return nil
			}
			if commit {
				fmt.Println(version.GitCommit)
				return nil
			}
			fmt.Printf("%s version %s\n", filepath.Base(os.Args[0]), version.Version)
			if version.GitCommit != "" {
				fmt.Printf("commit: %s\n", version.GitCommit)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&short, "short", false, "print just the version number")
	cmd.Flags().BoolVar(&commit, "commit", false, "print just the commit")

	return cmd
}
