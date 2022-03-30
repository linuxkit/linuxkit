package cache

import (
	"fmt"

	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/match"
	"github.com/google/go-containerregistry/pkg/v1/partial"
)

// matchPlatformsOSArch because match.Platforms rejects it if the provided
// v1.Platform has a variant of "" but the actual index has a specific one.
// This becomes an issue with arm64 vs arm64/v8. So this matches only on OS
// and Architecture.
func matchPlatformsOSArch(platforms ...v1.Platform) match.Matcher {
	return func(desc v1.Descriptor) bool {
		if desc.Platform == nil {
			return false
		}
		for _, platform := range platforms {
			if desc.Platform.OS == platform.OS && desc.Platform.Architecture == platform.Architecture {
				return true
			}
		}
		return false
	}
}

func (p *Provider) findImage(imageName, architecture string) (v1.Image, error) {
	root, err := p.FindRoot(imageName)
	if err != nil {
		return nil, err
	}
	img, err := root.Image()
	if err == nil {
		return img, nil
	}
	ii, err := root.ImageIndex()
	if err == nil {
		// we have the index, get the manifest that represents the manifest for the desired architecture
		platform := v1.Platform{OS: "linux", Architecture: architecture}
		images, err := partial.FindImages(ii, matchPlatformsOSArch(platform))
		if err != nil || len(images) < 1 {
			return nil, fmt.Errorf("error retrieving image %s for platform %v from cache: %v", imageName, platform, err)
		}
		return images[0], nil
	}
	return nil, fmt.Errorf("no image found for %s", imageName)
}

// FindDescriptor get the first descriptor pointed to by the image name
func (p *Provider) FindDescriptor(name string) (*v1.Descriptor, error) {
	index, err := p.cache.ImageIndex()
	// if there is no root index, we are broken
	if err != nil {
		return nil, fmt.Errorf("invalid image cache: %v", err)
	}

	descs, err := partial.FindManifests(index, match.Name(name))
	if err != nil {
		return nil, err
	}
	if len(descs) < 1 {
		return nil, nil
	}
	return &descs[0], nil
}
