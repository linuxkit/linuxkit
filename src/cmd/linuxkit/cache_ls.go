package main

import (
	"flag"

	cachepkg "github.com/linuxkit/linuxkit/src/cmd/linuxkit/cache"
	log "github.com/sirupsen/logrus"
)

func cacheList(args []string) {
	flags := flag.NewFlagSet("list", flag.ExitOnError)

	cacheDir := flags.String("cache", defaultLinuxkitCache(), "Directory for caching and finding cached image")

	if err := flags.Parse(args); err != nil {
		log.Fatal("Unable to parse args")
	}

	// list all of the images and content in the cache
	p, err := cachepkg.Get(*cacheDir)
	if err != nil {
		log.Fatalf("unable to read a local cache: %v", err)
	}
	images, err := cachepkg.ListImages(p)
	if err != nil {
		log.Fatalf("error reading image names: %v", err)
	}
	log.Printf("%-80s %s", "image name", "root manifest hash")
	for name, hash := range images {
		log.Printf("%-80s %s", name, hash)
	}
}
