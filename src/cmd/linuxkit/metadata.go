package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rn/iso9660wrap"
	log "github.com/sirupsen/logrus"
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

func metadataCreateUsage() {
	invoked := filepath.Base(os.Args[0])
	fmt.Printf("USAGE: %s metadata create [file.iso] [metadata]\n\n", invoked)

	fmt.Printf("'file.iso' is the file to create.\n")
	fmt.Printf("'metadata' will be written to '/config' in the ISO.\n")
	fmt.Printf("This is compatible with the linuxkit/metadata package\n")
}

func metadataCreate(args []string) {
	if len(args) != 2 {
		metadataCreateUsage()
		os.Exit(1)
	}
	switch args[0] {
	case "help", "-h", "-help", "--help":
		metadataCreateUsage()
		os.Exit(0)
	}

	isoImage := args[0]
	metadata := args[1]

	if err := WriteMetadataISO(isoImage, []byte(metadata)); err != nil {
		log.Fatal("Failed to write user data ISO: ", err)
	}
}

func metadataUsage() {
	invoked := filepath.Base(os.Args[0])
	fmt.Printf("USAGE: %s metadata COMMAND [options]\n\n", invoked)
	fmt.Printf("Commands:\n")
	fmt.Printf("  create      Create a metadata ISO\n")
}

func metadata(args []string) {
	if len(args) < 1 {
		metadataUsage()
		os.Exit(1)
	}
	switch args[0] {
	case "help", "-h", "-help", "--help":
		metadataUsage()
		os.Exit(0)
	}

	switch args[0] {
	case "create":
		metadataCreate(args[1:])
	default:
		fmt.Printf("%q is not a valid metadata command.\n\n", args[0])
		metadataUsage()
		os.Exit(1)
	}
}
