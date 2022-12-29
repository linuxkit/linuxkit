package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/version"
	"github.com/spf13/cobra"
)

func versionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "report the version of linuxkit",
		Long:  `Report the version of linuxkit.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("%s version %s\n", filepath.Base(os.Args[0]), version.Version)
			if version.GitCommit != "" {
				fmt.Printf("commit: %s\n", version.GitCommit)
			}
			return nil
		},
	}

	return cmd
}
