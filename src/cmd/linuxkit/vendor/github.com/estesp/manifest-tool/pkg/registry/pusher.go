package registry

import (
	"context"
	"encoding/json"

	"github.com/estesp/manifest-tool/pkg/store"
	"github.com/estesp/manifest-tool/pkg/types"

	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/remotes"
	"github.com/docker/distribution/manifest/manifestlist"
	"github.com/docker/distribution/reference"
	"github.com/opencontainers/go-digest"
	specs "github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Push performs the actions required to push content to the specified registry endpoint
func Push(m types.ManifestList, addedTags []string, ms *store.MemoryStore) (string, int, error) {
	// push manifest references to target ref (if required)
	baseRef := reference.TrimNamed(m.Reference)
	for _, man := range m.Manifests {
		if man.PushRef {
			ref, err := reference.WithDigest(baseRef, man.Descriptor.Digest)
			if err != nil {
				return "", 0, errors.Wrapf(err, "Error parsing reference for target manifest component push: %s", m.Reference.String())
			}
			err = push(ref, man.Descriptor, m.Resolver, ms)
			if err != nil {
				return "", 0, errors.Wrapf(err, "Error pushing target manifest component reference: %s", ref.String())
			}
			logrus.Infof("pushed manifest component reference (%s) to target namespace: %s", man.Descriptor.Digest.String(), ref.String())
		}
	}
	// build the manifest list/index entry to be pushed and save it in the content store
	desc, indexJSON, err := buildManifest(m)
	if err != nil {
		return "", 0, errors.Wrap(err, "Error creating manifest list/index JSON")
	}
	ms.Set(desc, indexJSON)

	if err := push(m.Reference, desc, m.Resolver, ms); err != nil {
		return "", 0, errors.Wrapf(err, "Error pushing manifest list/index to registry: %s", desc.Digest.String())
	}
	for _, tag := range addedTags {
		taggedRef, err := reference.WithTag(baseRef, tag)
		if err != nil {
			return "", 0, errors.Wrapf(err, "Error creating additional tag reference: %s", tag)
		}
		if err = push(taggedRef, desc, m.Resolver, ms); err != nil {
			return "", 0, errors.Wrapf(err, "Error pushing additional tag reference: %s", tag)
		}
	}
	return desc.Digest.String(), int(desc.Size), nil
}

func buildManifest(m types.ManifestList) (ocispec.Descriptor, []byte, error) {
	var (
		index     interface{}
		mediaType string
	)
	switch m.Type {
	case types.Docker:
		index = dockerManifestList(m.Manifests)
		mediaType = types.MediaTypeDockerSchema2ManifestList

	case types.OCI:
		index = ociIndex(m.Manifests)
		mediaType = ocispec.MediaTypeImageIndex
	}
	bytes, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return ocispec.Descriptor{}, []byte{}, err
	}
	desc := ocispec.Descriptor{
		Digest:    digest.FromBytes(bytes),
		MediaType: mediaType,
		Size:      int64(len(bytes)),
	}
	return desc, bytes, nil
}

func push(ref reference.Reference, desc ocispec.Descriptor, resolver remotes.Resolver, ms *store.MemoryStore) error {
	ctx := context.Background()
	pusher, err := resolver.Pusher(ctx, ref.String())
	if err != nil {
		return err
	}
	var wrapper func(images.Handler) images.Handler
	return remotes.PushContent(ctx, pusher, desc, ms, nil, wrapper)
}

func ociIndex(m []types.Manifest) ocispec.Index {
	index := ocispec.Index{
		Versioned: specs.Versioned{
			SchemaVersion: 2,
		},
	}
	for _, man := range m {
		index.Manifests = append(index.Manifests, man.Descriptor)
	}
	return index
}

func dockerManifestList(m []types.Manifest) manifestlist.ManifestList {
	ml := manifestlist.ManifestList{
		Versioned: manifestlist.SchemaVersion,
	}
	for _, man := range m {
		ml.Manifests = append(ml.Manifests, dockerConvert(man.Descriptor))
	}
	return ml
}

func dockerConvert(m ocispec.Descriptor) manifestlist.ManifestDescriptor {
	var md manifestlist.ManifestDescriptor
	md.Digest = m.Digest
	md.Size = m.Size
	md.MediaType = m.MediaType
	md.Platform.Architecture = m.Platform.Architecture
	md.Platform.OS = m.Platform.OS
	md.Platform.Variant = m.Platform.Variant
	md.Platform.OSFeatures = m.Platform.OSFeatures
	md.Platform.OSVersion = m.Platform.OSVersion
	return md
}
