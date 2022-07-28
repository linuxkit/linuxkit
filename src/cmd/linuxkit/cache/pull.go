package cache

import (
	"errors"
	"fmt"

	"github.com/containerd/containerd/reference"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/validate"
	lktspec "github.com/linuxkit/linuxkit/src/cmd/linuxkit/spec"
)

// ValidateImage given a reference, validate that it is complete. If not, pull down missing
// components as necessary. It also calculates the hash of each component.
func (p *Provider) ValidateImage(ref *reference.Spec, architecture string) (lktspec.ImageSource, error) {
	var (
		imageIndex v1.ImageIndex
		image      v1.Image
		imageName  = ref.String()
		desc       *v1.Descriptor
	)
	// next try the local cache
	root, err := p.FindRoot(imageName)
	if err == nil {
		img, err := root.Image()
		if err == nil {
			image = img
			if desc, err = partial.Descriptor(img); err != nil {
				return ImageSource{}, errors.New("image could not create valid descriptor")
			}
		} else {
			ii, err := root.ImageIndex()
			if err == nil {
				imageIndex = ii
				if desc, err = partial.Descriptor(ii); err != nil {
					return ImageSource{}, errors.New("index could not create valid descriptor")
				}
			}
		}
	}
	// three possibilities now:
	// - we did not find anything locally
	// - we found an index locally
	// - we found an image locally
	switch {
	case imageIndex == nil && image == nil:
		// we did not find it yet - either because we were told not to look locally,
		// or because it was not available - so get it from the remote
		return ImageSource{}, errors.New("no such image")
	case imageIndex != nil:
		// check that the index has a manifest for our arch
		im, err := imageIndex.IndexManifest()
		if err != nil {
			return ImageSource{}, fmt.Errorf("could not get index manifest: %v", err)
		}
		for _, m := range im.Manifests {
			if m.Platform != nil && m.Platform.Architecture == architecture && m.Platform.OS == linux {
				img, err := imageIndex.Image(m.Digest)
				if err != nil {
					return ImageSource{}, fmt.Errorf("unable to get image: %v", err)
				}
				if err := validate.Image(img); err != nil {
					return ImageSource{}, fmt.Errorf("invalid image: %s", err)
				}
				return p.NewSource(
					ref,
					architecture,
					desc,
				), nil
			}
		}
		return ImageSource{}, fmt.Errorf("index for %s did not contain image for platform linux/%s", imageName, architecture)
	case image != nil:
		// we found a local image, make sure it is up to date, and that it matches our platform
		if err := validate.Image(image); err != nil {
			return ImageSource{}, fmt.Errorf("invalid image, %s", err)
		}
		return p.NewSource(
			ref,
			architecture,
			desc,
		), nil
	}
	// if we made it to here, we had some strange error
	return ImageSource{}, errors.New("should not have reached this point, image index and image were both empty and not-empty")
}
