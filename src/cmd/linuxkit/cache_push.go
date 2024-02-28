package main

import (
	cachepkg "github.com/linuxkit/linuxkit/src/cmd/linuxkit/cache"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func cachePushCmd() *cobra.Command {
	var remoteName string
	cmd := &cobra.Command{
		Use:   "push",
		Short: "push images from the linuxkit cache",
		Long: `Push named images from the linuxkit cache to registry. Can provide short name, like linuxkit/kernel:6.6.13
		or nginx, or canonical name, like docker.io/library/nginx:latest.
		It is efficient, as blobs with the same content are not replaced.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			names := args
			for _, name := range names {
				fullname := util.ReferenceExpand(name)

				p, err := cachepkg.NewProvider(cacheDir)
				if err != nil {
					log.Fatalf("unable to read a local cache: %v", err)
				}

				if err := p.Push(fullname, remoteName, true); err != nil {
					log.Fatalf("unable to push image named %s: %v", name, err)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&remoteName, "remote-name", "", "Push it under a different name, e.g. push local image foo/bar:mine as baz/bee:yours. If blank, uses same local name.")

	return cmd
}
