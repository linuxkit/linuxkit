package cache

import (
	"github.com/containerd/containerd/v2/core/content"
)

// Store get content.Store referencing the cache
func (p *Provider) Store() (content.Store, error) {
	return p.store, nil
}
