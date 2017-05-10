package main

import (
	"fmt"
	"os"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
	"github.com/rneugeba/iso9660wrap"
)

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

	outfh, err := os.OpenFile(isoImage, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("Cannot create user data ISOs", "err", err)
	}
	defer outfh.Close()

	err = iso9660wrap.WriteBuffer(outfh, []byte(metadata), "config")
	if err != nil {
		log.Fatal("Cannot write user data ISO", "err", err)
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
