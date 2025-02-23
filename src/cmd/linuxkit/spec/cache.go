package spec

import (
	"io"

	"github.com/containerd/containerd/reference"
	"github.com/containerd/containerd/v2/core/content"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	imagespec "github.com/opencontainers/image-spec/specs-go/v1"
)

// CacheProvider interface for a provide of a cache.
type CacheProvider interface {
	// FindDescriptor find the descriptor pointed to by the reference in the cache.
	// ref is a valid reference, such as docker.io/library/alpine:3.15 or alpine@sha256:4edbd2beb5f78b1014028f4fbb99f3237d9561100b6881aabbf5acce2c4f9454
	// If both tag and digest are provided, will use digest exclusively.
	// Will expand to full names, so "alpine" becomes "docker.io/library/alpine:latest".
	// If none is found, returns nil Descriptor and no error.
	FindDescriptor(ref *reference.Spec) (*v1.Descriptor, error)
	// ImagePull takes an image name and pulls it from a registry to the cache. It should be
	// efficient and only write missing blobs, based on their content hash. If the ref already
	// exists in the cache, it should not pull anything, unless alwaysPull is set to true.
	ImagePull(ref *reference.Spec, platform []imagespec.Platform, alwaysPull bool) error
	// ImageInCache takes an image name and checks if it exists in the cache, including checking that the given
	// architecture is complete. Like ImagePull, it should be efficient and only write missing blobs, based on
	// their content hash.
	ImageInCache(ref *reference.Spec, trustedRef, architecture string) (bool, error)
	// ImageInRegistry takes an image name and checks if it exists in the registry.
	ImageInRegistry(ref *reference.Spec, trustedRef, architecture string) (bool, error)
	// IndexWrite takes an image name and creates an index for the descriptors to which it points.
	// Cache implementation determines whether it should pull missing blobs from a remote registry.
	// If the provided reference already exists and it is an index, updates the manifests in the
	// existing index.
	IndexWrite(ref *reference.Spec, descriptors ...v1.Descriptor) error
	// ImageLoad takes an OCI format image tar stream in the io.Reader and writes it to the cache. It should be
	// efficient and only write missing blobs, based on their content hash.
	ImageLoad(r io.Reader) ([]v1.Descriptor, error)
	// DescriptorWrite writes a descriptor to the cache index; it validates that it has a name
	// and replaces any existing one
	DescriptorWrite(ref *reference.Spec, descriptors v1.Descriptor) error
	// Push an image along with a multi-arch index from local cache to remote registry.
	// name is the name as referenced in the local cache, remoteName is the name to give it remotely.
	// If remoteName is empty, it is the same as name.
	// if withManifest defined will push a multi-arch manifest
	Push(name, remoteName string, withManifest, override bool) error
	// NewSource return an ImageSource for a specific ref and architecture in the cache.
	NewSource(ref *reference.Spec, platform *imagespec.Platform, descriptor *v1.Descriptor) ImageSource
	// GetContent returns an io.Reader to the provided content as is, given a specific digest. It is
	// up to the caller to validate it.
	GetContent(hash v1.Hash) (io.ReadCloser, error)
	// Store get content.Store referencing the cache
	Store() (content.Store, error)
}
