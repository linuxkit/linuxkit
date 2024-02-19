package main

import (
	"path/filepath"

	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/util"
	"github.com/spf13/cobra"
)

func defaultLinuxkitCache() string {
	lktDir := ".linuxkit"
	home := util.HomeDir()
	return filepath.Join(home, lktDir, "cache")
}

func cacheCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "cache",
		Short: "manage the linuxkit cache",
		Long:  `manage the linuxkit cache.`,
	}

	cmd.AddCommand(cacheCleanCmd())
	cmd.AddCommand(cacheRmCmd())
	cmd.AddCommand(cacheLsCmd())
	cmd.AddCommand(cacheExportCmd())
	cmd.AddCommand(cacheImportCmd())
	return cmd
}
