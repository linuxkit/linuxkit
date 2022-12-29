package main

import (
	"os"

	"github.com/rn/iso9660wrap"
	"github.com/spf13/cobra"
)

// WriteMetadataISO writes a metadata ISO file in a format usable by pkg/metadata
func WriteMetadataISO(path string, content []byte) error {
	outfh, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer outfh.Close()

	return iso9660wrap.WriteBuffer(outfh, content, "config")
}

func metadataCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "create an ISO with metadata",
		Long: `Create an ISO file with metadata in it.
		Provided metadata will be written to '/config' in the ISO.
		This is compatible with the linuxkit/metadata package.`,
		Args:    cobra.ExactArgs(2),
		Example: "linuxkit metadata create file.iso \"metadata\"",
		RunE: func(cmd *cobra.Command, args []string) error {
			isoImage := args[0]
			metadata := args[1]

			return WriteMetadataISO(isoImage, []byte(metadata))
		},
	}

	return cmd
}

func metadataCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "metadata",
		Short: "manage ISO metadata",
		Long:  `Manage ISO metadata.`,
	}

	cmd.AddCommand(metadataCreateCmd())

	return cmd
}
