package spec

import (
	"io"

	"github.com/containerd/containerd/reference"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

// CacheProvider interface for a provide of a cache.
type CacheProvider interface {
	// FindDescriptor get the first descriptor pointed to by the image name
	FindDescriptor(name string) (*v1.Descriptor, error)
	// ImagePull takes an image name and pulls it from a registry to the cache. It should be
	// efficient and only write missing blobs, based on their content hash. If the ref already
	// exists in the cache, it should not pull anything, unless alwaysPull is set to true.
	ImagePull(ref *reference.Spec, trustedRef, architecture string, alwaysPull bool) (ImageSource, error)
	// IndexWrite takes an image name and creates an index for the descriptors to which it points.
	// Cache implementation determines whether it should pull missing blobs from a remote registry.
	// If the provided reference already exists and it is an index, updates the manifests in the
	// existing index.
	IndexWrite(ref *reference.Spec, descriptors ...v1.Descriptor) (ImageSource, error)
	// ImageLoad takes an OCI format image tar stream in the io.Reader and writes it to the cache. It should be
	// efficient and only write missing blobs, based on their content hash.
	ImageLoad(ref *reference.Spec, architecture string, r io.Reader) (ImageSource, error)
	// DescriptorWrite writes a descriptor to the cache index; it validates that it has a name
	// and replaces any existing one
	DescriptorWrite(ref *reference.Spec, descriptors v1.Descriptor) (ImageSource, error)
	// Push push an image along with a multi-arch index from local cache to remote registry.
	Push(name string) error
	// NewSource return an ImageSource for a specific ref and architecture in the cache.
	NewSource(ref *reference.Spec, architecture string, descriptor *v1.Descriptor) ImageSource
}
