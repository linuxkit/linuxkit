package main

import (
	"io"
	"os"
	"runtime"

	"github.com/containerd/containerd/reference"
	cachepkg "github.com/linuxkit/linuxkit/src/cmd/linuxkit/cache"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func cacheExportCmd() *cobra.Command {
	var (
		arch       string
		outputFile string
		format     string
		tagName    string
	)
	cmd := &cobra.Command{
		Use:   "export",
		Short: "export individual images from the linuxkit cache",
		Long:  `Export individual images from the linuxkit cache. Supports exporting into multiple formats.`,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			names := args
			name := names[0]
			fullname := util.ReferenceExpand(name)

			p, err := cachepkg.NewProvider(cacheDir)
			if err != nil {
				log.Fatalf("unable to read a local cache: %v", err)
			}
			ref, err := reference.Parse(fullname)
			if err != nil {
				log.Fatalf("invalid image name %s: %v", name, err)
			}
			desc, err := p.FindDescriptor(&ref)
			if err != nil {
				log.Fatalf("unable to find image named %s: %v", name, err)
			}

			src := p.NewSource(&ref, arch, desc)
			var reader io.ReadCloser
			switch format {
			case "oci":
				fullTagName := fullname
				if tagName != "" {
					fullTagName = util.ReferenceExpand(tagName)
				}
				reader, err = src.V1TarReader(fullTagName)
			case "filesystem":
				reader, err = src.TarReader()
			default:
				log.Fatalf("requested unknown format %s: %v", name, err)
			}
			if err != nil {
				log.Fatalf("error getting reader for image %s: %v", name, err)
			}
			defer reader.Close()

			// try to write the output file
			var w io.Writer
			switch {
			case outputFile == "":
				log.Fatal("'outfile' flag is required")
			case outputFile == "-":
				w = os.Stdout
			default:
				f, err := os.OpenFile(outputFile, os.O_CREATE|os.O_RDWR, 0644)
				if err != nil {
					log.Fatalf("unable to open %s: %v", outputFile, err)
				}
				defer f.Close()
				w = f
			}

			_, err = io.Copy(w, reader)
			return err
		},
	}

	cmd.Flags().StringVar(&arch, "arch", runtime.GOARCH, "Architecture to resolve an index to an image, if the provided image name is an index")
	cmd.Flags().StringVar(&outputFile, "outfile", "", "Path to file to save output, '-' for stdout")
	cmd.Flags().StringVar(&format, "format", "oci", "export format, one of 'oci', 'filesystem'")
	cmd.Flags().StringVar(&tagName, "name", "", "override the provided image name in the exported tar file; useful only for format=oci")

	return cmd
}
