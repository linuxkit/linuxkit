package types

import (
	"github.com/containerd/containerd/remotes"
	"github.com/docker/distribution/reference"
	digest "github.com/opencontainers/go-digest"
)

// Request represents a registry reference and (optional) digest to a specific container manifest
type Request struct {
	reference  reference.Named
	digest     digest.Digest
	mediaTypes []string
	resolver   remotes.Resolver
}

// NewRequest creates a request from supplied image parameters
func NewRequest(ref reference.Named, digest digest.Digest, mediaTypes []string, resolver remotes.Resolver) *Request {
	return &Request{
		reference:  ref,
		digest:     digest,
		mediaTypes: mediaTypes,
		resolver:   resolver,
	}
}

// MediaTypes returns the media type string for this image
func (r *Request) MediaTypes() []string {
	return r.mediaTypes
}

// Reference returns the image reference as a `Named` object
func (r *Request) Reference() reference.Named {
	return r.reference
}

// Digest returns the image digesh hash of this image manifest
func (r *Request) Digest() digest.Digest {
	return r.digest
}

// Resolver returns the containerd remote's Docker resolver to use for the request
func (r *Request) Resolver() remotes.Resolver {
	return r.resolver
}
