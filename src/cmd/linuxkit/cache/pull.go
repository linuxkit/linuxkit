package cache

import (
	"errors"
	"fmt"

	"github.com/containerd/containerd/reference"
	"github.com/google/go-containerregistry/pkg/authn"
	namepkg "github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/match"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/validate"
	lktspec "github.com/linuxkit/linuxkit/src/cmd/linuxkit/spec"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/util"
	imagespec "github.com/opencontainers/image-spec/specs-go/v1"
	log "github.com/sirupsen/logrus"
)

const (
	unknown = "unknown"
)

// ValidateImage given a reference, validate that it is complete. If not, it will *not*
// pull down missing content. This function is network-free. If you wish to validate
// and fill in missing content, use PullImage.
// If the reference is to an index, it will check the index for a manifest for the given
// architecture, and any manifests that have no architecture at all. It will ignore manifests
// for other architectures. If no architecture is provided, it will validate all manifests.
// It also calculates the hash of each component.
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
		return ImageSource{}, &noReferenceError{reference: imageName}
	case imageIndex != nil:
		// check that the index has a manifest for our arch, as well as any non-arch-specific ones
		im, err := imageIndex.IndexManifest()
		if err != nil {
			return ImageSource{}, fmt.Errorf("could not get index manifest: %w", err)
		}
		var architectures = make(map[string]bool)
		// ignore only other architectures; manifest entries that have no architectures at all
		// are going to be additional metadata, so we need to check them
		for _, m := range im.Manifests {
			if m.Platform == nil || (m.Platform.Architecture == unknown && m.Platform.OS == unknown) {
				if err := validateManifestContents(imageIndex, m.Digest); err != nil {
					return ImageSource{}, fmt.Errorf("invalid image: %w", err)
				}
			}
			if architecture != "" && m.Platform.Architecture == architecture && m.Platform.OS == linux {
				if err := validateManifestContents(imageIndex, m.Digest); err != nil {
					return ImageSource{}, fmt.Errorf("invalid image: %w", err)
				}
				architectures[architecture] = true
			}
		}
		if architecture == "" || architectures[architecture] {
			return p.NewSource(
				ref,
				architecture,
				desc,
			), nil
		}
		return ImageSource{}, fmt.Errorf("index for %s did not contain image for platform linux/%s", imageName, architecture)
	case image != nil:
		// we found a local image, make sure it is up to date
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

// validateManifestContents given an index and a digest, validate that the contents of the
// manifest are a valid image. This function is network-free.
// The only validation it does is checks that all of the parts of the image exist.
func validateManifestContents(index v1.ImageIndex, digest v1.Hash) error {
	img, err := index.Image(digest)
	if err != nil {
		return fmt.Errorf("unable to get image: %w", err)
	}
	if _, err := img.ConfigFile(); err != nil {
		return fmt.Errorf("unable to get config: %w", err)
	}
	if _, err := img.Layers(); err != nil {
		return fmt.Errorf("unable to get layers: %w", err)
	}
	return nil
}

// Pull pull a reference, whether it points to an arch-specific image or to an index.
// If an index, optionally, try to pull its individual named references as well.
func (p *Provider) Pull(name string, withArchReferences bool) error {
	var (
		err error
	)
	fullname := util.ReferenceExpand(name, util.ReferenceWithTag())
	ref, err := namepkg.ParseReference(fullname)
	if err != nil {
		return err
	}
	v1ref, err := reference.Parse(ref.String())
	if err != nil {
		return err
	}

	// before we even try to push, let us see if it exists remotely
	remoteOptions := []remote.Option{remote.WithAuthFromKeychain(authn.DefaultKeychain)}

	desc, err := remote.Get(ref, remoteOptions...)
	if err != nil {
		return fmt.Errorf("error getting manifest for trusted image %s: %v", name, err)
	}

	// use the original image name in the annotation
	annotations := map[string]string{
		imagespec.AnnotationRefName: fullname,
	}

	// first attempt as an index
	ii, err := desc.ImageIndex()
	if err == nil {
		log.Debugf("ImageWrite retrieved %s is index, saving", fullname)

		if err := p.cache.WriteIndex(ii); err != nil {
			return fmt.Errorf("unable to write index: %v", err)
		}
		if _, err := p.DescriptorWrite(&v1ref, desc.Descriptor); err != nil {
			return fmt.Errorf("unable to write index descriptor to cache: %v", err)
		}
		if withArchReferences {
			im, err := ii.IndexManifest()
			if err != nil {
				return fmt.Errorf("unable to get IndexManifest: %v", err)
			}
			for _, m := range im.Manifests {
				if m.MediaType.IsImage() && m.Platform != nil && m.Platform.Architecture != unknown && m.Platform.OS != unknown {
					archSpecific := fmt.Sprintf("%s-%s", ref.String(), m.Platform.Architecture)
					archRef, err := reference.Parse(archSpecific)
					if err != nil {
						return fmt.Errorf("unable to parse arch-specific reference %s: %v", archSpecific, err)
					}
					if _, err := p.DescriptorWrite(&archRef, m); err != nil {
						return fmt.Errorf("unable to write index descriptor to cache: %v", err)
					}
				}
			}
		}
	} else {
		var im v1.Image
		// try an image
		im, err = desc.Image()
		if err != nil {
			return fmt.Errorf("provided image is neither an image nor an index: %s", name)
		}
		log.Debugf("ImageWrite retrieved %s is image, saving", fullname)
		if err = p.cache.ReplaceImage(im, match.Name(fullname), layout.WithAnnotations(annotations)); err != nil {
			return fmt.Errorf("unable to save image to cache: %v", err)
		}
	}

	return nil
}
