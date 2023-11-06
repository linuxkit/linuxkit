package cache

import (
	"fmt"
	"strings"

	"github.com/containerd/containerd/reference"
	v1 "github.com/google/go-containerregistry/pkg/v1"
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

// matchAllAnnotations returns a matcher that matches all annotations
func matchAllAnnotations(annotations map[string]string) match.Matcher {
	return func(desc v1.Descriptor) bool {
		if desc.Annotations == nil {
			return false
		}
		if len(annotations) == 0 {
			return true
		}
		for key, value := range annotations {
			if aValue, ok := desc.Annotations[key]; !ok || aValue != value {
				return false
			}
		}
		return true
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

func (p *Provider) findIndex(imageName string) (v1.ImageIndex, error) {
	root, err := p.FindRoot(imageName)
	if err != nil {
		return nil, err
	}
	ii, err := root.ImageIndex()
	if err != nil {
		return nil, fmt.Errorf("no image index found for %s", imageName)
	}
	return ii, nil
}

// FindDescriptor get the first descriptor pointed to by the image reference, whether tagged or digested
func (p *Provider) FindDescriptor(ref *reference.Spec) (*v1.Descriptor, error) {
	index, err := p.cache.ImageIndex()
	// if there is no root index, we are broken
	if err != nil {
		return nil, fmt.Errorf("invalid image cache: %v", err)
	}

	// parse the ref.Object to determine what it is, then search by that
	dig := ref.Digest()
	hash, err := v1.NewHash(dig.String())
	// if we had a valid hash, search by that
	if err == nil {
		// we had a valid hash, so we should search by it
		descs, err := partial.FindManifests(index, match.Digests(hash))
		if err != nil {
			return nil, err
		}
		if len(descs) > 0 {
			return &descs[0], nil
		}
		// we had a valid hash, but didn't find any descriptors
		return nil, nil
	}
	// no valid hash, try the tag
	tag := ref.Object
	// remove anything after an '@'
	n := strings.LastIndex(tag, "@")
	if n == 0 {
		return nil, fmt.Errorf("invalid tag, was not digest, yet began with '@': %s", tag)
	}
	if n >= 0 {
		tag = tag[:n]
	}
	descs, err := partial.FindManifests(index, match.Name(fmt.Sprintf("%s:%s", ref.Locator, tag)))
	if err != nil {
		return nil, err
	}
	if len(descs) > 0 {
		return &descs[0], nil
	}
	return nil, nil
}
