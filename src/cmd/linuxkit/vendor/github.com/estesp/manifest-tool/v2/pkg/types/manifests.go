package types

import (
	"github.com/containerd/containerd/remotes"
	"github.com/docker/distribution/reference"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// ManifestType specifies whether to use the OCI index media type and
// format or the Docker manifestList media type and format for the
// registry push operation.
type ManifestType int

const (
	// OCI is used to specify the "index" type
	OCI ManifestType = iota
	// Docker is used for the "manifestList" type
	Docker
)

// ManifestList represents the information necessary to assemble and
// push the right data to a registry to form a manifestlist or OCI index
// entry.
type ManifestList struct {
	Name      string
	Type      ManifestType
	Reference reference.Named
	Resolver  remotes.Resolver
	Manifests []Manifest
}

// Manifest is an ocispec.Descriptor of media type manifest (OCI or Docker)
// along with a boolean to help determine whether a reference to the manifest
// must be pushed to the target (manifest list) repo location before finalizing
// the manifest list push operation.
type Manifest struct {
	Descriptor ocispec.Descriptor
	PushRef    bool
}
