package cache

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/containerd/containerd/reference"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	imagespec "github.com/opencontainers/image-spec/specs-go/v1"
)

// ImageSource a source for an image in the OCI distribution cache.
// Implements a moby.ImageSource.
type ImageSource struct {
	ref          *reference.Spec
	cache        layout.Path
	architecture string
}

// NewSource return an ImageSource for a specific ref and architecture in the given
// cache directory.
func NewSource(ref *reference.Spec, dir string, architecture string) ImageSource {
	p, _ := Get(dir)
	return ImageSource{
		ref:          ref,
		cache:        p,
		architecture: architecture,
	}
}

// Config return the imagespec.ImageConfig for the given source. Resolves to the
// architecture, if necessary.
func (c ImageSource) Config() (imagespec.ImageConfig, error) {
	imageName := c.ref.String()
	image, err := findImage(c.cache, imageName, c.architecture)
	if err != nil {
		return imagespec.ImageConfig{}, err
	}

	configFile, err := image.ConfigFile()
	if err != nil {
		return imagespec.ImageConfig{}, fmt.Errorf("unable to get image OCI ConfigFile: %v", err)
	}
	// because the other parts expect OCI go-spec structs, not google/go-containerregistry structs,
	// the easiest way to do this is to convert via json
	configJSON, err := json.Marshal(configFile.Config)
	if err != nil {
		return imagespec.ImageConfig{}, fmt.Errorf("unable to convert image config to json: %v", err)
	}
	var ociConfig imagespec.ImageConfig
	err = json.Unmarshal(configJSON, &ociConfig)
	return ociConfig, err
}

// TarReader return an io.ReadCloser to read the filesystem contents of the image,
// as resolved to the provided architecture.
func (c ImageSource) TarReader() (io.ReadCloser, error) {
	imageName := c.ref.String()

	// get a reference to the image
	image, err := findImage(c.cache, imageName, c.architecture)
	if err != nil {
		return nil, err
	}

	return mutate.Extract(image), nil
}
