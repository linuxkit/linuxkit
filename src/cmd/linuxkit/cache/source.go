package cache

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/containerd/containerd/reference"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	lktspec "github.com/linuxkit/linuxkit/src/cmd/linuxkit/spec"
	imagespec "github.com/opencontainers/image-spec/specs-go/v1"
)

// ImageSource a source for an image in the OCI distribution cache.
// Implements a spec.ImageSource.
type ImageSource struct {
	ref          *reference.Spec
	provider     *Provider
	architecture string
	descriptor   *v1.Descriptor
}

// NewSource return an ImageSource for a specific ref and architecture in the given
// cache directory.
func (p *Provider) NewSource(ref *reference.Spec, architecture string, descriptor *v1.Descriptor) lktspec.ImageSource {
	return ImageSource{
		ref:          ref,
		provider:     p,
		architecture: architecture,
		descriptor:   descriptor,
	}
}

// Config return the imagespec.ImageConfig for the given source. Resolves to the
// architecture, if necessary.
func (c ImageSource) Config() (imagespec.ImageConfig, error) {
	imageName := c.ref.String()
	image, err := c.provider.findImage(imageName, c.architecture)
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
	image, err := c.provider.findImage(imageName, c.architecture)
	if err != nil {
		return nil, err
	}

	return mutate.Extract(image), nil
}

// V1TarReader return an io.ReadCloser to read the image as a v1 tarball
func (c ImageSource) V1TarReader(overrideName string) (io.ReadCloser, error) {
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
	image, err := c.provider.findImage(imageName, c.architecture)
	if err != nil {
		return nil, err
	}
	// convert the writer to a reader
	r, w := io.Pipe()
	go func() {
		defer w.Close()
		tarball.Write(refName, image, w)
	}()
	return r, nil
}

// Descriptor return the descriptor of the image.
func (c ImageSource) Descriptor() *v1.Descriptor {
	return c.descriptor
}
