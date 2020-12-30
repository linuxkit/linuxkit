package cache

import (
	"errors"

	"github.com/containerd/containerd/reference"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/validate"
)

// ValidateImage given a reference, validate that it is complete. If not, pull down missing
// components as necessary.
func ValidateImage(ref *reference.Spec, cacheDir, architecture string) (ImageSource, error) {
	var (
		imageIndex v1.ImageIndex
		image      v1.Image
		imageName  = ref.String()
	)
	// next try the local cache
	root, err := FindRoot(cacheDir, imageName)
	if err == nil {
		img, err := root.Image()
		if err == nil {
			image = img
		} else {
			ii, err := root.ImageIndex()
			if err == nil {
				imageIndex = ii
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
		// we found a local index, just make sure it is up to date and, if not, download it
		if err := validate.Index(imageIndex); err == nil {
			return NewSource(
				ref,
				cacheDir,
				architecture,
			), nil
		}
		return ImageSource{}, errors.New("invalid index")
	case image != nil:
		// we found a local image, just make sure it is up to date
		if err := validate.Image(image); err == nil {
			return NewSource(
				ref,
				cacheDir,
				architecture,
			), nil
		}
		return ImageSource{}, errors.New("invalid image")
	}
	// if we made it to here, we had some strange error
	return ImageSource{}, errors.New("should not have reached this point, image index and image were both empty and not-empty")
}
