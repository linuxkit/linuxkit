package cache

import (
	"github.com/google/go-containerregistry/pkg/v1/layout"
)

// Provider cache implementation of cacheProvider
type Provider struct {
	cache layout.Path
}

// NewProvider create a new CacheProvider based in the provided directory
func NewProvider(dir string) (*Provider, error) {
	p, err := Get(dir)
	if err != nil {
		return nil, err
	}
	return &Provider{p}, nil
}
