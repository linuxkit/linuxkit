package registry

import (
	"context"
	"encoding/json"

	ccontent "github.com/containerd/containerd/content"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/estesp/manifest-tool/v2/pkg/store"
	"github.com/estesp/manifest-tool/v2/pkg/types"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// Fetch uses a registry (distribution spec) API to retrieve a specific image manifest from a registry
func Fetch(ctx context.Context, cs *store.MemoryStore, req *types.Request) (ocispec.Descriptor, error) {

	resolver := req.Resolver()

	// Retrieve manifest from registry
	name, desc, err := resolver.Resolve(ctx, req.Reference().String())
	if err != nil {
		return ocispec.Descriptor{}, err
	}
	fetcher, err := resolver.Fetcher(ctx, name)
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	appendDistSrcLabelHandler, err := docker.AppendDistributionSourceLabel(cs, req.Reference().String())
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	handlers := []images.Handler{
		remotes.FetchHandler(cs, fetcher),
		nonLayerChildHandler(cs),
		appendDistSrcLabelHandler,
	}
	// This traverses the OCI descriptor to fetch the image and store it into the local store initialized above.
	// All content hashes are verified in this step
	if err := images.Dispatch(ctx, images.Handlers(handlers...), nil, desc); err != nil {
		return ocispec.Descriptor{}, err
	}
	return desc, nil
}

// nonLayerChildHandler returns the immediate children of content described by the descriptor, skipping layers
// and any other non-manifest/config descriptors. This code is copied and modified (to remove layer retrieval)
// from the "images.Children" handler in containerd
func nonLayerChildHandler(provider ccontent.Provider) images.HandlerFunc {
	return func(ctx context.Context, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		var descs []ocispec.Descriptor
		switch desc.MediaType {
		case types.MediaTypeDockerSchema2Manifest, ocispec.MediaTypeImageManifest:
			p, err := ccontent.ReadBlob(ctx, provider, desc)
			if err != nil {
				return nil, err
			}
			var manifest ocispec.Manifest
			if err := json.Unmarshal(p, &manifest); err != nil {
				return nil, err
			}

			descs = append(descs, manifest.Config)
		case types.MediaTypeDockerSchema2ManifestList, ocispec.MediaTypeImageIndex:
			p, err := ccontent.ReadBlob(ctx, provider, desc)
			if err != nil {
				return nil, err
			}
			var index ocispec.Index
			if err := json.Unmarshal(p, &index); err != nil {
				return nil, err
			}

			descs = append(descs, index.Manifests...)
		case ocispec.MediaTypeImageLayer, ocispec.MediaTypeImageLayerGzip, types.MediaTypeDockerTarGzipLayer:
			// we want to save the descriptor info about layers in our content store
			// in case we are going to handle push of a manifest list (will need to handle
			// have the details for doing blob mounting on manifest list/index push)
			cs, ok := provider.(*store.MemoryStore)
			if ok {
				// a slight hack to get the info about layers without the actual layer content
				// into our content store
				cs.Set(desc, []byte{})
			}
		default:
			// if we aren't at a manifest or index/manifestlist then we can stop walking
			return nil, nil

		}
		return descs, nil
	}
}
