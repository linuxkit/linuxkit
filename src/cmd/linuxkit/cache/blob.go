package cache

import (
	"io"

	v1 "github.com/google/go-containerregistry/pkg/v1"
)

func (p *Provider) GetContent(hash v1.Hash) (io.ReadCloser, error) {
	return p.cache.Blob(hash)
}
