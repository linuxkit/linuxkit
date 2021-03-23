package spec

import (
	"io"

	"github.com/google/go-containerregistry/pkg/v1"
	imagespec "github.com/opencontainers/image-spec/specs-go/v1"
)

// ImageSource interface to an image. It can have its config read, and a its containers
// can be read via an io.ReadCloser tar stream.
type ImageSource interface {
	Config() (imagespec.ImageConfig, error)
	TarReader() (io.ReadCloser, error)
	Descriptor() *v1.Descriptor
}
