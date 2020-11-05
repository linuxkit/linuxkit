package moby

import (
	"fmt"

	"github.com/containerd/containerd/reference"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/cache"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/docker"
)

// imagePull pull an image from the OCI registry to the cache.
// If the image root already is in the cache, use it, unless
// the option pull is set to true.
// if alwaysPull, then do not even bother reading locally
func imagePull(ref *reference.Spec, alwaysPull bool, trust bool, cacheDir string, dockerCache bool, architecture string) (ImageSource, error) {
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
		if image, err := cache.ValidateImage(ref, cacheDir, architecture); err == nil {
			return image, nil
		}
	}

	// if we made it here, we either did not have the image, or it was incomplete
	return imageLayoutWrite(cacheDir, ref, architecture, trust)
}

// imageLayoutWrite takes an image name and pulls it down, writing it locally
func imageLayoutWrite(cacheDir string, ref *reference.Spec, architecture string, trust bool) (ImageSource, error) {
	image := ref.String()
	var (
		trustedName string
	)
	if trust {
		// get trusted reference
		trustedRef, err := TrustedReference(image)
		if err != nil {
			return nil, fmt.Errorf("Trusted pull for %s failed: %v", ref, err)
		}
		trustedName = trustedRef.String()
	}
	return cache.ImageWrite(cacheDir, ref, trustedName, architecture)
}
