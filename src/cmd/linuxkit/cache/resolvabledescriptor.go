package cache

import (
	"fmt"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/match"
	"github.com/google/go-containerregistry/pkg/v1/partial"
)

/*
 The entire below section is in the process of being upstreamed to
 github.com/google/go-containerregistry
*/

// ResolvableDescriptor an item that can resolve to a v1.Image or a v1.ImageIndex
type ResolvableDescriptor interface {
	Image() (v1.Image, error)
	ImageIndex() (v1.ImageIndex, error)
	Digest() (v1.Hash, error)
}
type layoutImage struct {
	img v1.Image
}

func (l layoutImage) Image() (v1.Image, error) {
	return l.img, nil
}
func (l layoutImage) ImageIndex() (v1.ImageIndex, error) {
	return nil, fmt.Errorf("not an ImageIndex")
}
func (l layoutImage) Digest() (v1.Hash, error) {
	return l.img.Digest()
}

type layoutIndex struct {
	idx v1.ImageIndex
}

func (l layoutIndex) Image() (v1.Image, error) {
	return nil, fmt.Errorf("not an Image")
}
func (l layoutIndex) ImageIndex() (v1.ImageIndex, error) {
	return l.idx, nil
}
func (l layoutIndex) Digest() (v1.Hash, error) {
	return l.idx.Digest()
}

// FindRoot find the root ResolvableDescriptor, representing an Image or Index, for
// a given imageName.
func (p *Provider) FindRoot(imageName string) (ResolvableDescriptor, error) {
	rootIndex, desc, err := p.rootDescriptor(imageName)
	if err != nil {
		return nil, err
	}
	switch {
	case desc.MediaType.IsImage():
		img, err := rootIndex.Image(desc.Digest)
		if err != nil {
			return nil, fmt.Errorf("resolving local image %s (%s): %w", imageName, desc.Digest, err)
		}
		return layoutImage{img: img}, nil
	case desc.MediaType.IsIndex():
		idx, err := rootIndex.ImageIndex(desc.Digest)
		if err != nil {
			return nil, fmt.Errorf("resolving local index %s (%s): %w", imageName, desc.Digest, err)
		}
		return layoutIndex{idx: idx}, nil
	default:
		return nil, fmt.Errorf("local reference %s (%s) has unsupported media type %s", imageName, desc.Digest, desc.MediaType)
	}
}

// rootDescriptor distinguishes an absent tag from a tag with unreadable content.
func (p *Provider) rootDescriptor(imageName string) (v1.ImageIndex, *v1.Descriptor, error) {
	rootIndex, err := p.Index()
	if err != nil {
		return nil, nil, fmt.Errorf("reading cache index for %s: %w", imageName, err)
	}
	descs, err := partial.FindManifests(rootIndex, match.Name(imageName))
	if err != nil {
		return nil, nil, fmt.Errorf("finding local reference %s: %w", imageName, err)
	}
	if len(descs) == 0 {
		return nil, nil, fmt.Errorf("reference %s is not present in cache index", imageName)
	}
	return rootIndex, &descs[0], nil
}
