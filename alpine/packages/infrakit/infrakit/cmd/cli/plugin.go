package main

import (
	"fmt"
	"github.com/docker/infrakit/discovery"
	"github.com/spf13/cobra"
)

func pluginCommand(plugins func() discovery.Plugins) *cobra.Command {

	cmd := &cobra.Command{
		Use:   "plugin",
		Short: "Manage plugins",
	}

	var quiet bool
	ls := cobra.Command{
		Use:   "ls",
		Short: "List available plugins",
		RunE: func(c *cobra.Command, args []string) error {
			entries, err := plugins().List()
			if err != nil {
				return err
			}

			if !quiet {
				fmt.Printf("%-20s\t%-s\n", "NAME", "LISTEN")
			}
			for k, v := range entries {
				fmt.Printf("%-20s\t%-s\n", k, v.String())
			}

			return nil
		},
	}
	ls.Flags().BoolVarP(&quiet, "quiet", "q", false, "Print rows without column headers")

	cmd.AddCommand(&ls)

	return cmd
}
