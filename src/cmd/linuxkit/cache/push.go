package cache

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	namepkg "github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/match"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/google/go-containerregistry/pkg/v1/validate"

	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/registry"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/util"
	log "github.com/sirupsen/logrus"
)

// Push push an image along with a multi-arch index.
// name is the name as referenced in the local cache, remoteName is the name to give it remotely.
// If remoteName is empty, it is the same as name.
// If withArchSpecificTags is true, it will push all arch-specific images in the index, each as
// their own tag with the same name as the index, but with the architecture appended, e.g.
// image:foo will have image:foo-amd64, image:foo-arm64, image:foo-riscv64, etc.
func (p *Provider) Push(name, remoteName string, withArchSpecificTags, override bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultPushTimeout)
	defer cancel()
	return p.PushWithContext(ctx, name, remoteName, withArchSpecificTags, override)
}

// PushWithContext uses the caller's deadline for all registry operations.
func (p *Provider) PushWithContext(ctx context.Context, name, remoteName string, withArchSpecificTags, override bool) error {
	return p.PushWithOptions(ctx, name, remoteName, withArchSpecificTags, override, PushOptions{})
}

// PushWithOptions controls presentation without changing the CacheProvider interface.
func (p *Provider) PushWithOptions(ctx context.Context, name, remoteName string, withArchSpecificTags, override bool, settings PushOptions) (result error) {
	if err := settings.Validate(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("starting push of %s: %w", name, err)
	}
	var (
		err     error
		options []remote.Option
	)
	if remoteName == "" {
		remoteName = name
	}
	ref, err := namepkg.ParseReference(remoteName)
	if err != nil {
		return err
	}
	observer := newPushObserver()
	observer.progressMode = settings.Progress
	if settings.TraceUpload {
		observer.progressMode = PushProgressPlain
		trace, err := newUploadTrace(name, remoteName, settings.TraceWriter, settings.TraceInterval)
		if err != nil {
			return fmt.Errorf("starting upload trace: %w", err)
		}
		observer.trace = trace
		ctx = context.WithValue(ctx, uploadTraceContextKey{}, trace)
		defer func() {
			if err := trace.Close(result); err != nil {
				result = errors.Join(result, fmt.Errorf("writing upload diagnostics: %w", err))
			}
		}()
	}
	baseTransport := remote.DefaultTransport.(*http.Transport).Clone()
	defer baseTransport.CloseIdleConnections()
	options = append(options, remote.WithAuthFromKeychain(authn.DefaultKeychain),
		remote.WithContext(ctx), remote.WithTransport(pushTransport{RoundTripper: baseTransport, observer: observer}))

	fmt.Printf("Pushing local %s as %s\n", name, remoteName)

	// check if it already exists, unless override is explicit
	if !override {
		if _, err := getPushRemote(ctx, ref, options...); err == nil {
			log.Infof("image %s already exists in the registry, skipping", remoteName)
			return nil
		} else if !isPushNotFound(err) {
			return err
		}
	}

	// if we made it this far, either we had a specific override, or we do not have the image remotely

	// Check the local closure before merging remote content or starting uploads.
	// Remote-only descriptors retained by AppendIndex need not exist locally.
	if err := p.validatePushContents(name); err != nil {
		return err
	}

	// do we even have the given one?
	root, err := p.FindRoot(name)
	if err != nil {
		return err
	}

	img, err1 := root.Image()
	ii, err2 := root.ImageIndex()
	// before we even try to push, let us see if it exists remotely
	remoteOptions := options

	switch {
	case err1 == nil:
		dig, err := img.Digest()
		if err != nil {
			return fmt.Errorf("could not get digest for local image %s: %w", name, err)
		}
		desc, err := getPushRemote(ctx, ref, remoteOptions...)
		if err != nil && !isPushNotFound(err) {
			return err
		}
		if err == nil && desc != nil && dig == desc.Digest {
			fmt.Printf("%s image already available on remote registry, skipping push\n", remoteName)
			return nil
		}
		log.Debugf("pushing image %s as %s", name, remoteName)
		if err := observer.discover(img); err != nil {
			return fmt.Errorf("listing blobs for %s: %w", name, err)
		}
		if err := pushWithProgress(ctx, "image upload to "+remoteName, observer, options, func(opts ...remote.Option) error {
			return remote.Write(ref, observedImage{Image: img, observer: observer}, opts...)
		}); err != nil {
			return fmt.Errorf("uploading local image %s to %s: %w", name, remoteName, err)
		}
		fmt.Printf("Pushed image %s\n", name)
	case err2 == nil:
		dig, err := ii.Digest()
		if err != nil {
			return fmt.Errorf("could not get digest for index %s: %w", name, err)
		}
		manifest, err := ii.IndexManifest()
		if err != nil {
			return fmt.Errorf("could not read images in index: %w", err)
		}

		// get the existing image, if any
		desc, err := getPushRemote(ctx, ref, remoteOptions...)
		if err != nil && !isPushNotFound(err) {
			return err
		}
		if err == nil && desc != nil {
			if dig == desc.Digest {
				fmt.Printf("%s index already available on remote registry, skipping push\n", remoteName)
				return nil
			}
			// we have a different index, need to cross-reference and only override relevant stuff
			remoteIndex, err := desc.ImageIndex()
			if err == nil && remoteIndex != nil {
				ii, err = util.AppendIndex(ii, remoteIndex)
				if err != nil {
					return fmt.Errorf("could not append remote index to local index: %w", err)
				}
			}

			// clean up the index. Some platforms, like Docker's sboms, add extra manifests that have platform unknown/unknown
			// and reference existing images in the manifest by a specific annotation. Make sure there are none that reference something
			// that does not exist in the index. This could be an old artifact froma previous push that got overridden.
			ii, err = cleanDanglingReferences(ii)
			if err != nil {
				return fmt.Errorf("could not clean up index %s: %w", name, err)
			}
		}
		log.Debugf("pushing local index %s as %s", name, remoteName)
		if err := observer.discover(ii); err != nil {
			return fmt.Errorf("listing blobs for %s: %w", name, err)
		}
		// this is an index, so we not only want to write the index, but tags for each arch-specific image in it
		if err := pushWithProgress(ctx, "index upload to "+remoteName, observer, options, func(opts ...remote.Option) error {
			return remote.WriteIndex(ref, observedIndex{pushImageIndex: ii, observer: observer}, opts...)
		}); err != nil {
			return fmt.Errorf("uploading local index %s to %s: %w", name, remoteName, err)
		}
		fmt.Printf("Pushed index %s\n", name)
		if withArchSpecificTags {
			fmt.Printf("pushing individual images in the index %s\n", name)
			for _, m := range manifest.Manifests {
				if m.Platform == nil || m.Platform.Architecture == "" {
					continue
				}
				archTag := fmt.Sprintf("%s-%s", remoteName, m.Platform.Architecture)
				tag, err := namepkg.NewTag(archTag)
				if err != nil {
					return fmt.Errorf("could not create a valid arch-specific tag %s: %w", archTag, err)
				}
				img, err := p.cache.Image(m.Digest)
				if err != nil {
					// it might not have existed, so we can add it locally
					// use the original image name in the annotation
					desc := m.DeepCopy()
					if err := p.DescriptorWrite(archTag, *desc); err != nil {
						return fmt.Errorf("error appending descriptor for %s to layout index: %w", archTag, err)
					}
					img, err = p.cache.Image(m.Digest)
					if err != nil {
						return fmt.Errorf("could not find or create arch-specific image for %s: %w", archTag, err)
					}
				}
				if err := validate.Image(img); err != nil {
					// skip arch we did not build/pull locally
					log.Debugf("could not validate arch-specific image for %s: %v", archTag, err)
					continue
				}
				log.Debugf("pushing image %s", tag)
				if err := pushWithProgress(ctx, "creating architecture tag "+archTag, observer, options, func(opts ...remote.Option) error {
					return remote.Tag(tag, img, opts...)
				}); err != nil {
					return err
				}
			}
		} else {
			fmt.Printf("Skipping push of individual images in the index %s\n", name)
		}
	default:
		return fmt.Errorf("name %s unknown in cache", name)
	}

	return nil
}

// cleanDanglingReferences removes any dangling references from the index.
// If any have a tag that points to a manifest not in the index, it will remove it.
func cleanDanglingReferences(ii v1.ImageIndex) (v1.ImageIndex, error) {
	var (
		digests  = make(map[v1.Hash]bool)
		toRemove []v1.Hash
	)
	// first, record all of the valid digests
	existingManifests, err := ii.IndexManifest()
	if err != nil {
		return nil, fmt.Errorf("could not read index manifest: %w", err)
	}
	for _, m := range existingManifests.Manifests {
		if m.Platform == nil || m.Platform.Architecture == "" || m.Platform.Architecture == unknown {
			continue
		}
		digests[m.Digest] = true
	}
	// next make sure each item has a valid digest
	for _, m := range existingManifests.Manifests {
		if m.Platform == nil || m.Platform.Architecture != unknown || m.Platform.OS != unknown {
			continue
		}
		// get the annotations
		if m.Annotations == nil {
			continue
		}

		referenced, ok := m.Annotations[util.AnnotationDockerReferenceDigest]
		if !ok || referenced == "" {
			continue
		}
		// check if the referenced digest is in the index
		digest, err := v1.NewHash(referenced)
		if err != nil {
			return nil, fmt.Errorf("could not parse hash %s: %w", referenced, err)
		}
		// if it is not in the index, we need to remove this manifest
		if _, ok := digests[digest]; !ok {
			toRemove = append(toRemove, m.Digest)
		}
	}
	ii = mutate.RemoveManifests(ii, match.Digests(toRemove...))
	return ii, nil
}

// Only a missing reference should be treated as a reason to proceed with upload.
func isPushNotFound(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var responseError *transport.Error
	return errors.As(err, &responseError) && responseError.StatusCode == http.StatusNotFound
}
func getPushRemote(ctx context.Context, ref namepkg.Reference, options ...remote.Option) (*remote.Descriptor, error) {
	started := time.Now()
	var err error
	done := traceFromContext(ctx).operation("registry_lookup")
	defer func() { done(err) }()
	desc, err := registry.GetRemote().Get(ref, options...)
	if err == nil {
		return desc, nil
	}
	if ctx.Err() != nil {
		err = errors.Join(ctx.Err(), err)
	}
	return nil, &pushOperationError{label: "registry lookup for " + ref.Name(), progress: pushProgress{started: started, lastChange: started, finished: time.Now()}, cause: err}
}
