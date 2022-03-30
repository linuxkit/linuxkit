package cache

import (
	"github.com/google/go-containerregistry/pkg/v1/layout"
	imagespec "github.com/opencontainers/image-spec/specs-go/v1"
)

// ListImages list the named images and their root digests from a layout.Path
func ListImages(p layout.Path) (map[string]string, error) {
	ii, err := p.ImageIndex()
	if err != nil {
		return nil, err
	}
	index, err := ii.IndexManifest()
	if err != nil {
		return nil, err
	}
	names := map[string]string{}
	for _, i := range index.Manifests {
		if i.Annotations == nil {
			continue
		}
		if name, ok := i.Annotations[imagespec.AnnotationRefName]; ok {
			names[name] = i.Digest.String()
		}
	}
	return names, nil
}
