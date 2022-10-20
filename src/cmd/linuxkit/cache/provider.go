package cache

import (
	"path/filepath"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/content/local"
	"github.com/google/go-containerregistry/pkg/v1/layout"
)

// Provider cache implementation of cacheProvider
type Provider struct {
	cache layout.Path
	store content.Store
	blobs string
}

// NewProvider create a new CacheProvider based in the provided directory
func NewProvider(dir string) (*Provider, error) {
	p, err := Get(dir)
	if err != nil {
		return nil, err
	}
	store, err := local.NewStore(dir)
	if err != nil {
		return nil, err
	}
	return &Provider{
		cache: p,
		store: store,
		blobs: filepath.Join(dir, "cache", "blobs"),
	}, nil
}
