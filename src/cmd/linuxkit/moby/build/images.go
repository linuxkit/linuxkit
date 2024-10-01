package build

import (
	"github.com/containerd/containerd/reference"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/cache"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/docker"
	lktspec "github.com/linuxkit/linuxkit/src/cmd/linuxkit/spec"
	imagespec "github.com/opencontainers/image-spec/specs-go/v1"
)

// imageSource given an image ref, get a handle on the image so it can be used as a source for its configuration
// and layers. If the image root already is in the cache, use it.
// If not in cache, pull it down from the OCI registry.
// Optionally can look in docker image cache first, before falling back to linuxkit cache and OCI registry.
// Optionally can be told to alwaysPull, in which case it always pulls from the OCI registry.
// Always works for a single architecture, as we are referencing a specific image.
func imageSource(ref *reference.Spec, alwaysPull bool, cacheDir string, dockerCache bool, platform imagespec.Platform) (lktspec.ImageSource, error) {
	// several possibilities:
	// - alwaysPull: try to pull it down from the registry to linuxkit cache, then fail
	// - !alwaysPull && dockerCache: try to read it from docker, then try linuxkit cache, then try to pull from registry, then fail
	// - !alwaysPull && !dockerCache: try linuxkit cache, then try to pull from registry, then fail
	// first, try docker, if that is available
	if !alwaysPull && dockerCache {
		if err := docker.HasImage(ref, platform.Architecture); err == nil {
			return docker.NewSource(ref), nil
		}
		// docker is not required, so any error - image not available, no docker, whatever - just gets ignored
	}

	// get a reference to the local cache; we either will find the ref there or will pull to it
	c, err := cache.NewProvider(cacheDir)
	if err != nil {
		return nil, err
	}

	// if we made it here, we either did not have the image, or it was incomplete
	if err := c.ImagePull(ref, []imagespec.Platform{platform}, alwaysPull); err != nil {
		return nil, err
	}
	desc, err := c.FindDescriptor(ref)
	if err != nil {
		return nil, err
	}
	return c.NewSource(
		ref,
		&platform,
		desc,
	), nil
}

// indexSource given an image ref, get a handle on the index so it can be used as a source for its underlying images.
// If the index root already is in the cache, use it.
// If not in cache, pull it down from the OCI registry.
// Optionally can look in docker image cache first, before falling back to linuxkit cache and OCI registry.
// Optionally can be told to alwaysPull, in which case it always pulls from the OCI registry.
// Can provide architectures to list which ones to limit, or leave empty for all available.
func indexSource(ref *reference.Spec, alwaysPull bool, cacheDir string, platforms []imagespec.Platform) (lktspec.IndexSource, error) {
	// get a reference to the local cache; we either will find the ref there or will pull to it
	c, err := cache.NewProvider(cacheDir)
	if err != nil {
		return nil, err
	}

	// if we made it here, we either did not have the image, or it was incomplete
	if err := c.ImagePull(ref, platforms, alwaysPull); err != nil {
		return nil, err
	}
	desc, err := c.FindDescriptor(ref)
	if err != nil {
		return nil, err
	}
	return c.NewIndexSource(
		ref,
		desc,
		platforms,
	), nil
}
