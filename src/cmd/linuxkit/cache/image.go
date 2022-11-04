package cache

import (
	imagespec "github.com/opencontainers/image-spec/specs-go/v1"
)

// ListImages list the named images and their root digests from a layout.Path
func ListImages(dir string) (map[string]string, error) {
	p, err := NewProvider(dir)
	if err != nil {
		return nil, err
	}
	return p.List()
}

func (p *Provider) List() (map[string]string, error) {
	ii, err := p.cache.ImageIndex()
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
