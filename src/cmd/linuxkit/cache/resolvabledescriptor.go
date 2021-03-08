package cache

import (
	"fmt"

	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/layout"
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

type layoutIndex struct {
	idx v1.ImageIndex
}

func (l layoutIndex) Image() (v1.Image, error) {
	return nil, fmt.Errorf("not an Image")
}
func (l layoutIndex) ImageIndex() (v1.ImageIndex, error) {
	return l.idx, nil
}

// FindRoot find the root ResolvableDescriptor, representing an Image or Index, for
// a given imageName.
func FindRoot(dir, imageName string) (ResolvableDescriptor, error) {
	p, err := Get(dir)
	if err != nil {
		return nil, err
	}
	return findRootFromLayout(p, imageName)
}

func findRootFromLayout(p layout.Path, imageName string) (ResolvableDescriptor, error) {
	matcher := match.Name(imageName)
	rootIndex, err := p.ImageIndex()
	// of there is no root index, we are broken
	if err != nil {
		return nil, fmt.Errorf("invalid image cache: %v", err)
	}

	// first try the root tag as an image itself
	images, err := partial.FindImages(rootIndex, matcher)
	if err == nil && len(images) > 0 {
		// if we found the root tag as an image, just use it
		return layoutImage{img: images[0]}, nil
	}
	// we did not find the root tag as an image, it is an index, get the index
	indexes, err := partial.FindIndexes(rootIndex, matcher)
	if err == nil && len(indexes) >= 1 {
		return layoutIndex{idx: indexes[0]}, nil
	}
	return nil, fmt.Errorf("could not find image or index for %s", imageName)
}
