package spec

import (
	"io"

	"github.com/containerd/containerd/reference"
	"github.com/google/go-containerregistry/pkg/v1"
)

// CacheProvider interface for a provide of a cache.
type CacheProvider interface {
	FindDescriptor(name string) (*v1.Descriptor, error)
	ImagePull(ref *reference.Spec, trustedRef, architecture string, alwaysPull bool) (ImageSource, error)
	IndexWrite(ref *reference.Spec, descriptors ...v1.Descriptor) (ImageSource, error)
	ImageLoad(ref *reference.Spec, architecture string, r io.Reader) (ImageSource, error)
	DescriptorWrite(ref *reference.Spec, descriptors v1.Descriptor) (ImageSource, error)
	Push(name string) error
	NewSource(ref *reference.Spec, architecture string, descriptor *v1.Descriptor) ImageSource
}
