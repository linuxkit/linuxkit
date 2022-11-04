package main

import (
	"flag"
	"fmt"

	cachepkg "github.com/linuxkit/linuxkit/src/cmd/linuxkit/cache"
	log "github.com/sirupsen/logrus"
)

func cacheRm(args []string) {
	flags := flag.NewFlagSet("rm", flag.ExitOnError)

	cacheDir := flagOverEnvVarOverDefaultString{def: defaultLinuxkitCache(), envVar: envVarCacheDir}
	flags.Var(&cacheDir, "cache", fmt.Sprintf("Directory for caching and finding cached image, overrides env var %s", envVarCacheDir))
	publishedOnly := flags.Bool("published-only", false, "Only remove the specified images if linuxkit can confirm at the time of running have been published to the registry")

	if err := flags.Parse(args); err != nil {
		log.Fatal("Unable to parse args")
	}

	if flags.NArg() == 0 {
		log.Fatal("Please specify at least one image to remove")
	}

	imageNames := flags.Args()

	// did we limit to published only?

	// list all of the images and content in the cache
	p, err := cachepkg.NewProvider(cacheDir.String())
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
	removeImagesFromCache(images, p, *publishedOnly)
}
