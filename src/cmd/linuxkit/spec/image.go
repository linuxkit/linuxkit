package spec

import (
	"io"

	"github.com/google/go-containerregistry/pkg/v1"
	imagespec "github.com/opencontainers/image-spec/specs-go/v1"
)

// ImageSource interface to an image. It can have its config read, and a its containers
// can be read via an io.ReadCloser tar stream.
type ImageSource interface {
	// Config get the config for the image
	Config() (imagespec.ImageConfig, error)
	// TarReader get the flattened filesystem of the image as a tar stream/
	TarReader() (io.ReadCloser, error)
	// Descriptor get the v1.Descriptor of the image
	Descriptor() *v1.Descriptor
	// V1TarReader get the image as v1 tarball, also compatibel with `docker load`
	V1TarReader() (io.ReadCloser, error)
}
