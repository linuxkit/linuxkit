package cache

import (
	"errors"
	"fmt"
	"strings"

	"github.com/containerd/containerd/v2/pkg/reference"
	"github.com/google/go-containerregistry/pkg/authn"
	namepkg "github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/validate"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/registry"
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
func (p *Provider) ValidateImage(ref *reference.Spec, platforms []imagespec.Platform) (lktspec.ImageSource, error) {
	var (
		imageIndex      v1.ImageIndex
		image           v1.Image
		imageName       = ref.String()
		desc            *v1.Descriptor
		platformMessage = platformMessageGenerator(platforms)
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
		var (
			targetPlatforms = make(map[string]bool)
			foundPlatforms  = make(map[string]bool)
		)
		for _, plat := range platforms {
			pString := platformString(plat)
			targetPlatforms[pString] = false
			foundPlatforms[pString] = false
		}
		// ignore only other architectures; manifest entries that have no architectures at all
		// are going to be additional metadata, so we need to check them
		for _, m := range im.Manifests {
			if m.Platform == nil || (m.Platform.Architecture == unknown && m.Platform.OS == unknown) {
				if err := validateManifestContents(imageIndex, m.Digest); err != nil {
					return ImageSource{}, fmt.Errorf("invalid image: %w", err)
				}
			}
			// go through each target platform, and see if this one matched. If it did, mark the target as
			for _, plat := range platforms {
				if plat.Architecture == m.Platform.Architecture && plat.OS == m.Platform.OS &&
					(plat.Variant == "" || plat.Variant == m.Platform.Variant) {
					targetPlatforms[platformString(plat)] = true
					break
				}
			}
		}

		if len(platforms) == 0 {
			return p.NewSource(
				ref,
				nil,
				desc,
			), nil
		}
		// we have cycled through all of the manifests, let's check if we have all of the platforms
		var missing []string
		for plat, found := range targetPlatforms {
			if !found {
				missing = append(missing, plat)
			}
		}

		if len(missing) == 0 {
			return p.NewSource(
				ref,
				nil,
				desc,
			), nil
		}
		return ImageSource{}, fmt.Errorf("index for %s did not contain image for platforms %s", imageName, strings.Join(missing, ", "))
	case image != nil:
		if len(platforms) > 1 {
			return ImageSource{}, fmt.Errorf("image %s is not a multi-arch image, but asked for %s", imageName, platformMessage)
		}
		// we found a local image, make sure it is up to date
		if err := validate.Image(image); err != nil {
			return ImageSource{}, fmt.Errorf("invalid image, %s", err)
		}
		return p.NewSource(
			ref,
			&platforms[0],
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

	desc, err := registry.GetRemote().Get(ref, remoteOptions...)
	if err != nil {
		return fmt.Errorf("error getting manifest for trusted image %s: %v", name, err)
	}

	// lock the cache so we can write to it
	if err := p.Lock(); err != nil {
		return fmt.Errorf("unable to lock cache for writing: %v", err)
	}
	defer p.Unlock()

	// first attempt as an index
	ii, err := desc.ImageIndex()
	if err == nil {
		log.Debugf("ImageWrite retrieved %s is index, saving", fullname)

		if err := p.cache.WriteIndex(ii); err != nil {
			return fmt.Errorf("unable to write index: %v", err)
		}
		if err := p.DescriptorWrite(v1ref.String(), desc.Descriptor); err != nil {
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
					if _, err := reference.Parse(archSpecific); err != nil {
						return fmt.Errorf("unable to parse arch-specific reference %s: %v", archSpecific, err)
					}
					if err := p.DescriptorWrite(archSpecific, m); err != nil {
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
		if err = p.cache.WriteImage(im); err != nil {
			return fmt.Errorf("unable to save image to cache: %v", err)
		}
		if err = p.DescriptorWrite(fullname, desc.Descriptor); err != nil {
			return fmt.Errorf("unable to write updated descriptor to cache: %v", err)
		}
	}

	return nil
}
