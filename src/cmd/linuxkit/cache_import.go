package main

import (
	"io"
	"os"

	cachepkg "github.com/linuxkit/linuxkit/src/cmd/linuxkit/cache"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func cacheImportCmd() *cobra.Command {
	var (
		inputFile string
	)
	cmd := &cobra.Command{
		Use:   "import",
		Short: "import individual images to the linuxkit cache",
		Long: `Import individual images from tar file to the linuxkit cache.
		Can provide the file on the command-line or via stdin with filename '-'.
		
		Example:
		linuxkit cache import myimage.tar
		cat myimage.tar | linuxkit cache import -

		Tarfile format must be the OCI v1 file format, see https://github.com/opencontainers/image-spec/blob/main/image-layout.md
		`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			paths := args
			infile := paths[0]

			p, err := cachepkg.NewProvider(cacheDir)
			if err != nil {
				log.Fatalf("unable to read a local cache: %v", err)
			}

			var reader io.ReadCloser
			if inputFile == "-" {
				reader = os.Stdin
			} else {
				f, err := os.Open(infile)
				if err != nil {
					log.Fatalf("unable to open %s: %v", infile, err)
				}
				defer f.Close()
				reader = f
			}
			defer reader.Close()

			if _, err := p.ImageLoad(reader); err != nil {
				log.Fatalf("unable to load image: %v", err)
			}

			return err
		},
	}

	return cmd
}
