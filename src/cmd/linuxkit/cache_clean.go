package main

import (
	"flag"
	"fmt"
	"os"

	namepkg "github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	cachepkg "github.com/linuxkit/linuxkit/src/cmd/linuxkit/cache"
	log "github.com/sirupsen/logrus"
)

func cacheClean(args []string) {
	flags := flag.NewFlagSet("clean", flag.ExitOnError)

	cacheDir := flagOverEnvVarOverDefaultString{def: defaultLinuxkitCache(), envVar: envVarCacheDir}
	flags.Var(&cacheDir, "cache", fmt.Sprintf("Directory for caching and finding cached image, overrides env var %s", envVarCacheDir))
	publishedOnly := flags.Bool("published-only", false, "Only clean images that linuxkit can confirm at the time of running have been published to the registry")

	if err := flags.Parse(args); err != nil {
		log.Fatal("Unable to parse args")
	}

	// did we limit to published only?
	if !*publishedOnly {
		if err := os.RemoveAll(cacheDir.String()); err != nil {
			log.Fatalf("Unable to clean cache %s: %v", cacheDir, err)
		}
		log.Infof("Cache emptied: %s", cacheDir)
		return
	}

	// list all of the images and content in the cache
	p, err := cachepkg.NewProvider(cacheDir.String())
	if err != nil {
		log.Fatalf("unable to read a local cache: %v", err)
	}
	images, err := p.List()

	if err != nil {
		log.Fatalf("error reading image names: %v", err)
	}
	removeImagesFromCache(images, p, *publishedOnly)
}

// removeImagesFromCache removes images from the cache.
func removeImagesFromCache(images map[string]string, p *cachepkg.Provider, publishedOnly bool) {
	// check each image in the registry. If it exists, remove it here.
	for name, hash := range images {
		if publishedOnly {
			ref, err := namepkg.ParseReference(name)
			if err != nil {
				continue
			}
			desc, err := remote.Get(ref)
			if err != nil {
				log.Debugf("image %s not found in remote registry or error, leaving in cache: %v", name, err)
				fmt.Fprintf(os.Stderr, "image %s not found in remote registry, leaving in cache", name)
				continue
			}
			if desc == nil {
				fmt.Fprintf(os.Stderr, "image %s not found in remote registry, leaving in cache", name)
				continue
			}
			if desc.Digest.String() != hash {
				fmt.Fprintf(os.Stderr, "image %s has mismatched hashes, cache %s vs remote registry %s, leaving in cache", name, hash, desc.Digest.String())
				continue
			}
		}
		// we have a match, remove it
		fmt.Fprintf(os.Stderr, "removing image %s from cache", name)
		if err := p.Remove(name); err != nil {
			log.Warnf("Unable to remove image %s: %v", name, err)
		}
	}
}
