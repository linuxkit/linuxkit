package cache

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"

	"github.com/containerd/containerd/v2/pkg/reference"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/spec"
	lktspec "github.com/linuxkit/linuxkit/src/cmd/linuxkit/spec"
	imagespec "github.com/opencontainers/image-spec/specs-go/v1"
)

// IndexSource a source for an image in the OCI distribution cache.
// Implements a spec.ImageSource.
type IndexSource struct {
	ref        *reference.Spec
	provider   *Provider
	descriptor *v1.Descriptor
	platforms  []imagespec.Platform
}

// NewIndexSource return an IndexSource for a specific ref in the given
// cache directory.
func (p *Provider) NewIndexSource(ref *reference.Spec, descriptor *v1.Descriptor, platforms []imagespec.Platform) lktspec.IndexSource {
	return IndexSource{
		ref:        ref,
		provider:   p,
		descriptor: descriptor,
		platforms:  platforms,
	}
}

// Config return the imagespec.ImageConfig for the given source. Resolves to the
// architecture, if necessary.
func (c IndexSource) Image(platform imagespec.Platform) (spec.ImageSource, error) {
	imageName := c.ref.String()
	index, err := c.provider.findIndex(imageName)
	if err != nil {
		return nil, err
	}
	manifests, err := index.IndexManifest()
	if err != nil {
		return nil, err
	}
	for _, manifest := range manifests.Manifests {
		if manifest.Platform != nil && manifest.Platform.Architecture == platform.Architecture && manifest.Platform.OS == platform.OS {
			return c.provider.NewSource(c.ref, &platform, &manifest), nil
		}
	}
	return nil, fmt.Errorf("no manifest found for platform %q", platform)
}

// OCITarReader return an io.ReadCloser to read the image as a v1 tarball whose contents match OCI v1 layout spec
func (c IndexSource) OCITarReader(overrideName string) (io.ReadCloser, error) {
	imageName := c.ref.String()
	saveName := imageName
	if overrideName != "" {
		saveName = overrideName
	}
	refName, err := name.ParseReference(saveName)
	if err != nil {
		return nil, fmt.Errorf("error parsing image name: %v", err)
	}
	// get a reference to the image
	index, err := c.provider.findIndex(c.ref.String())
	if err != nil {
		return nil, err
	}
	// convert the writer to a reader
	r, w := io.Pipe()
	go func() {
		defer w.Close()
		tw := tar.NewWriter(w)
		defer tw.Close()
		if err := writeLayoutHeader(tw); err != nil {
			_ = w.CloseWithError(err)
			return
		}

		manifests, err := index.IndexManifest()
		if err != nil {
			_ = w.CloseWithError(err)
			return
		}
		// for each manifest, write the manifest blob, then go through each manifest and find the image for it
		// and write its blobs
		for _, manifest := range manifests.Manifests {
			// if we restricted this image source to certain platforms, we should only write those
			if len(c.platforms) > 0 {
				found := false
				for _, platform := range c.platforms {
					if platform.Architecture == manifest.Platform.Architecture && platform.OS == manifest.Platform.OS &&
						(platform.Variant == "" || platform.Variant == manifest.Platform.Variant) {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}
			switch manifest.MediaType {
			case types.OCIManifestSchema1, types.DockerManifestSchema2:
				// this is an image manifest
				image, err := index.Image(manifest.Digest)
				if err != nil {
					_ = w.CloseWithError(err)
					return
				}
				if err := writeLayoutImage(tw, image); err != nil {
					_ = w.CloseWithError(err)
					return
				}
			}
		}

		// write the index directly as a blob
		indexSize, err := index.Size()
		if err != nil {
			_ = w.CloseWithError(err)
			return
		}
		indexDigest, err := index.Digest()
		if err != nil {
			_ = w.CloseWithError(err)
			return
		}
		indexBytes, err := index.RawManifest()
		if err != nil {
			_ = w.CloseWithError(err)
			return
		}
		if err := writeLayoutBlob(tw, indexDigest.Hex, indexSize, bytes.NewReader(indexBytes)); err != nil {
			_ = w.CloseWithError(err)
			return
		}

		desc := v1.Descriptor{
			MediaType: types.OCIImageIndex,
			Size:      indexSize,
			Digest:    indexDigest,
			Annotations: map[string]string{
				imagespec.AnnotationRefName: refName.String(),
			},
		}
		if err := writeLayoutIndex(tw, desc); err != nil {
			_ = w.CloseWithError(err)
			return
		}
	}()
	return r, nil
}

// Descriptor return the descriptor of the index.
func (c IndexSource) Descriptor() *v1.Descriptor {
	return c.descriptor
}
