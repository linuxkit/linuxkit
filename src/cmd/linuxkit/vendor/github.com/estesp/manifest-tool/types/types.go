package types

import (
	"github.com/docker/distribution/manifest/manifestlist"
	"github.com/docker/docker/api/types/container"
)

// ImageInspect holds information about an image in a registry
type ImageInspect struct {
	Size            int64
	MediaType       string
	Tag             string
	Digest          string
	RepoTags        []string
	Comment         string
	Created         string
	ContainerConfig *container.Config
	DockerVersion   string
	Author          string
	Config          *container.Config
	Architecture    string
	Os              string
	OSVersion       string
	OSFeatures      []string
	Layers          []string
	References      []string
	Platform        manifestlist.PlatformSpec
	CanonicalJSON   []byte
}

// YAMLInput represents the YAML format input to the pushml
// command.
type YAMLInput struct {
	Image     string
	Tags      []string
	Manifests []ManifestEntry
}

// ManifestEntry represents an entry in the list of manifests to
// be combined into a manifest list, provided via the YAML input
type ManifestEntry struct {
	Image    string
	Platform manifestlist.PlatformSpec
}

// AuthInfo holds information about how manifest-tool should connect and authenticate to the docker registry
type AuthInfo struct {
	Username  string
	Password  string
	DockerCfg string
}
