package main

import (
	"flag"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
)

func cacheClean(args []string) {
	flags := flag.NewFlagSet("clean", flag.ExitOnError)

	cacheDir := flagOverEnvVarOverDefaultString{def: defaultLinuxkitCache(), envVar: envVarCacheDir}
	flags.Var(&cacheDir, "cache", fmt.Sprintf("Directory for caching and finding cached image, overrides env var %s", envVarCacheDir))

	if err := flags.Parse(args); err != nil {
		log.Fatal("Unable to parse args")
	}

	if err := os.RemoveAll(cacheDir.String()); err != nil {
		log.Fatalf("Unable to clean cache %s: %v", cacheDir, err)
	}
	log.Infof("Cache cleaned: %s", cacheDir)
}
