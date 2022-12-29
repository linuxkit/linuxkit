package main

import (
	cachepkg "github.com/linuxkit/linuxkit/src/cmd/linuxkit/cache"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func cacheLsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ls",
		Short: "list images in the linuxkit cache",
		Long:  `List images in the linuxkit cache.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// list all of the images and content in the cache
			images, err := cachepkg.ListImages(cacheDir)
			if err != nil {
				log.Fatalf("error reading image names: %v", err)
			}
			log.Printf("%-80s %s", "image name", "root manifest hash")
			for name, hash := range images {
				log.Printf("%-80s %s", name, hash)
			}
			return nil
		},
	}
	return cmd
}
