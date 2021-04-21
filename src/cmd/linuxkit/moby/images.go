package moby

import (
	"github.com/containerd/containerd/reference"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/cache"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/docker"
	lktspec "github.com/linuxkit/linuxkit/src/cmd/linuxkit/spec"
)

// imagePull pull an image from the OCI registry to the cache.
// If the image root already is in the cache, use it, unless
// the option pull is set to true.
// if alwaysPull, then do not even bother reading locally
func imagePull(ref *reference.Spec, alwaysPull bool, cacheDir string, dockerCache bool, architecture string) (lktspec.ImageSource, error) {
	// several possibilities:
	// - alwaysPull: try to pull it down from the registry to linuxkit cache, then fail
	// - !alwaysPull && dockerCache: try to read it from docker, then try linuxkit cache, then try to pull from registry, then fail
	// - !alwaysPull && !dockerCache: try linuxkit cache, then try to pull from registry, then fail
	// first, try docker, if that is available
	if !alwaysPull && dockerCache {
		if err := docker.HasImage(ref); err == nil {
			return docker.NewSource(ref), nil
		}
		// docker is not required, so any error - image not available, no docker, whatever - just gets ignored
	}

	// next try the local cache
	if !alwaysPull {
		c, err := cache.NewProvider(cacheDir)
		if err != nil {
			return nil, err
		}
		if image, err := c.ValidateImage(ref, architecture); err == nil {
			return image, nil
		}
	}

	// if we made it here, we either did not have the image, or it was incomplete
	return imageLayoutWrite(cacheDir, ref, architecture)
}

// imageLayoutWrite takes an image name and pulls it down, writing it locally
func imageLayoutWrite(cacheDir string, ref *reference.Spec, architecture string) (lktspec.ImageSource, error) {
	image := ref.String()
	c, err := cache.NewProvider(cacheDir)
	if err != nil {
		return nil, err
	}
	return c.ImagePull(ref, image, architecture)
}
