package types

import (
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

const (
	// MediaTypeDockerSchema2Manifest is the Docker v2.2 schema media type for a manifest object
	MediaTypeDockerSchema2Manifest = "application/vnd.docker.distribution.manifest.v2+json"
	// MediaTypeDockerSchema2ManifestList is the Docker v2.2 schema media type for a manifest list object
	MediaTypeDockerSchema2ManifestList = "application/vnd.docker.distribution.manifest.list.v2+json"
	// MediaTypeDockerTarLayer is the Docker schema media type for a tar filesystem layer
	MediaTypeDockerTarLayer = "application/vnd.docker.image.rootfs.diff.tar.gzip"
	// MediaTypeDockerTarGzipLayer is the Docker schema media type for a tar+gzip filesystem layer
	MediaTypeDockerTarGzipLayer = "application/vnd.docker.image.rootfs.foreign.diff.tar.gzip"
)

// Image struct handles Windows support extensions to OCI spec
type Image struct {
	ocispec.Image
	OSVersion  string   `json:"os.version,omitempty"`
	OSFeatures []string `json:"os.features,omitempty"`
}
