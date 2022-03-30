package main

import (
	"flag"
	"os"

	log "github.com/sirupsen/logrus"
)

func cacheClean(args []string) {
	flags := flag.NewFlagSet("clean", flag.ExitOnError)

	cacheDir := flags.String("cache", defaultLinuxkitCache(), "Directory for caching and finding cached image")

	if err := flags.Parse(args); err != nil {
		log.Fatal("Unable to parse args")
	}

	if err := os.RemoveAll(*cacheDir); err != nil {
		log.Fatalf("Unable to clean cache %s: %v", *cacheDir, err)
	}
	log.Infof("Cache cleaned: %s", *cacheDir)
}
