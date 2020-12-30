package cache

import (
	"fmt"

	"github.com/containerd/containerd/reference"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/match"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	imagespec "github.com/opencontainers/image-spec/specs-go/v1"
)

// ImageWrite takes an image name and pulls it down, writing it locally. It should be
// efficient and only write missing blobs, based on their content hash.
func ImageWrite(dir string, ref *reference.Spec, trustedRef, architecture string) (ImageSource, error) {
	p, err := Get(dir)
	if err != nil {
		return ImageSource{}, err
	}
	image := ref.String()
	pullImageName := image
	remoteOptions := []remote.Option{remote.WithAuthFromKeychain(authn.DefaultKeychain)}
	if trustedRef != "" {
		pullImageName = trustedRef
	}
	remoteRef, err := name.ParseReference(pullImageName)
	if err != nil {
		return ImageSource{}, fmt.Errorf("invalid image name %s: %v", pullImageName, err)
	}

	desc, err := remote.Get(remoteRef, remoteOptions...)
	if err != nil {
		return ImageSource{}, fmt.Errorf("error getting manifest for trusted image %s: %v", pullImageName, err)
	}

	// use the original image name in the annotation
	annotations := map[string]string{
		imagespec.AnnotationRefName: image,
	}

	// first attempt as an index
	ii, err := desc.ImageIndex()
	if err == nil {
		err = p.ReplaceIndex(ii, match.Name(image), layout.WithAnnotations(annotations))
	} else {
		var im v1.Image
		// try an image
		im, err = desc.Image()
		if err != nil {
			return ImageSource{}, fmt.Errorf("provided image is neither an image nor an index: %s", image)
		}
		err = p.ReplaceImage(im, match.Name(image), layout.WithAnnotations(annotations))
	}
	if err != nil {
		return ImageSource{}, fmt.Errorf("unable to save image to cache: %v", err)
	}
	return NewSource(
		ref,
		dir,
		architecture,
	), nil
}
