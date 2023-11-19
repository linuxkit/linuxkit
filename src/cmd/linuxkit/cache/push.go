package cache

import (
	"fmt"

	"github.com/google/go-containerregistry/pkg/authn"
	namepkg "github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/validate"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/util"
	imagespec "github.com/opencontainers/image-spec/specs-go/v1"
	log "github.com/sirupsen/logrus"
)

// Push push an image along with a multi-arch index.
func (p *Provider) Push(name string, withManifest bool) error {
	var (
		err     error
		options []remote.Option
	)
	ref, err := namepkg.ParseReference(name)
	if err != nil {
		return err
	}

	fmt.Printf("Pushing %s\n", name)
	// do we even have the given one?
	root, err := p.FindRoot(name)
	if err != nil {
		return err
	}
	options = append(options, remote.WithAuthFromKeychain(authn.DefaultKeychain))

	img, err1 := root.Image()
	ii, err2 := root.ImageIndex()
	// before we even try to push, let us see if it exists remotely
	remoteOptions := []remote.Option{remote.WithAuthFromKeychain(authn.DefaultKeychain)}

	switch {
	case err1 == nil:
		dig, err := img.Digest()
		if err != nil {
			return fmt.Errorf("could not get digest for image %s: %v", name, err)
		}
		desc, err := remote.Get(ref, remoteOptions...)
		if err == nil && desc != nil && dig == desc.Digest {
			fmt.Printf("%s image already available on remote registry, skipping push", name)
			return nil
		}
		log.Debugf("pushing image %s", name)
		if err := remote.Write(ref, img, options...); err != nil {
			return err
		}
		fmt.Printf("Pushed image %s\n", name)
	case err2 == nil:
		dig, err := ii.Digest()
		if err != nil {
			return fmt.Errorf("could not get digest for index %s: %v", name, err)
		}
		manifest, err := ii.IndexManifest()
		if err != nil {
			return fmt.Errorf("could not read images in index: %v", err)
		}

		// get the existing image, if any
		desc, err := remote.Get(ref, remoteOptions...)
		if err == nil && desc != nil {
			if dig == desc.Digest {
				fmt.Printf("%s index already available on remote registry, skipping push", name)
				return nil
			}
			// we have a different index, need to cross-reference and only override relevant stuff
			remoteIndex, err := desc.ImageIndex()
			if err == nil && remoteIndex != nil {
				ii, err = util.AppendIndex(ii, remoteIndex)
				if err != nil {
					return fmt.Errorf("could not append remote index to local index: %v", err)
				}
			}
		}
		log.Debugf("pushing index %s", name)
		// this is an index, so we not only want to write the index, but tags for each arch-specific image in it
		if err := remote.WriteIndex(ref, ii, options...); err != nil {
			return err
		}
		fmt.Printf("Pushed index %s\n", name)
		log.Debugf("pushing individual images in the index %s", name)
		for _, m := range manifest.Manifests {
			if m.Platform == nil || m.Platform.Architecture == "" {
				continue
			}
			archTag := fmt.Sprintf("%s-%s", name, m.Platform.Architecture)
			tag, err := namepkg.NewTag(archTag)
			if err != nil {
				return fmt.Errorf("could not create a valid arch-specific tag %s: %v", archTag, err)
			}
			img, err := p.cache.Image(m.Digest)
			if err != nil {
				// it might not have existed, so we can add it locally
				// use the original image name in the annotation
				desc := m.DeepCopy()
				if desc.Annotations == nil {
					desc.Annotations = map[string]string{}
				}
				desc.Annotations[imagespec.AnnotationRefName] = archTag
				if err := p.cache.AppendDescriptor(*desc); err != nil {
					return fmt.Errorf("error appending descriptor for %s to layout index: %v", archTag, err)
				}
				img, err = p.cache.Image(m.Digest)
				if err != nil {
					return fmt.Errorf("could not find or create arch-specific image for %s: %v", archTag, err)
				}
			}
			if err := validate.Image(img); err != nil {
				// skip arch we did not build/pull locally
				log.Debugf("could not validate arch-specific image for %s: %v", archTag, err)
				continue
			}
			log.Debugf("pushing image %s", tag)
			if err := remote.Tag(tag, img, options...); err != nil {
				return fmt.Errorf("error creating tag %s: %v", archTag, err)
			}
		}
	default:
		return fmt.Errorf("name %s unknown in cache", name)
	}

	return nil
}
