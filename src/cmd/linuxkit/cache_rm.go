package main

import (
	cachepkg "github.com/linuxkit/linuxkit/src/cmd/linuxkit/cache"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func cacheRmCmd() *cobra.Command {
	var (
		publishedOnly bool
	)
	cmd := &cobra.Command{
		Use:   "rm",
		Short: "remove individual images from the linuxkit cache",
		Long:  `Remove individual images from the linuxkit cache.`,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			imageNames := args

			// did we limit to published only?

			// list all of the images and content in the cache
			p, err := cachepkg.NewProvider(cacheDir)
			if err != nil {
				log.Fatalf("unable to read a local cache: %v", err)
			}
			images := map[string]string{}
			for _, imageName := range imageNames {
				desc, err := p.FindRoot(imageName)
				if err != nil {
					log.Fatalf("error reading image %s: %v", imageName, err)
				}
				dig, err := desc.Digest()
				if err != nil {
					log.Fatalf("error reading digest for image %s: %v", imageName, err)
				}
				images[imageName] = dig.String()
			}
			removeImagesFromCache(images, p, publishedOnly)
			return nil
		},
	}

	cmd.Flags().BoolVar(&publishedOnly, "published-only", false, "Only clean images that linuxkit can confirm at the time of running have been published to the registry")

	return cmd
}
