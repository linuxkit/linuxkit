package main

import (
	cachepkg "github.com/linuxkit/linuxkit/src/cmd/linuxkit/cache"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func cachePullCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pull",
		Short: "pull images to the linuxkit cache from registry",
		Long: `Pull named images from their registry to the linuxkit cache. Can provide short name, like linuxkit/kernel:6.6.13
		or nginx, or canonical name, like docker.io/library/nginx:latest. Will be saved into cache as canonical.
		Will replace in cache if found. Blobs with the same content are not replaced.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			names := args
			for _, name := range names {
				fullname := util.ReferenceExpand(name, util.ReferenceWithTag())

				p, err := cachepkg.NewProvider(cacheDir)
				if err != nil {
					log.Fatalf("unable to read a local cache: %v", err)
				}

				if err := p.Pull(fullname, true); err != nil {
					log.Fatalf("unable to push image named %s: %v", name, err)
				}
			}
			return nil
		},
	}

	return cmd
}
