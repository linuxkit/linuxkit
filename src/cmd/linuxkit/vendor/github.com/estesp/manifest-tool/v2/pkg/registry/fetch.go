package registry

import (
	"context"

	"github.com/containerd/containerd/remotes"
	"github.com/docker/distribution/reference"
	"github.com/estesp/manifest-tool/v2/pkg/store"
	"github.com/estesp/manifest-tool/v2/pkg/types"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func FetchDescriptor(resolver remotes.Resolver, memoryStore *store.MemoryStore, imageRef reference.Named) (ocispec.Descriptor, error) {
	return Fetch(context.Background(), memoryStore, types.NewRequest(imageRef, "", allMediaTypes(), resolver))
}

func allMediaTypes() []string {
	return []string{
		types.MediaTypeDockerSchema2Manifest,
		types.MediaTypeDockerSchema2ManifestList,
		ocispec.MediaTypeImageManifest,
		ocispec.MediaTypeImageIndex,
	}
}
