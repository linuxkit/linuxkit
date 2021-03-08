package cache

import (
	"fmt"

	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/layout"
)

// Get get or initialize the cache
func Get(cache string) (layout.Path, error) {
	// initialize the cache path if needed
	p, err := layout.FromPath(cache)
	if err != nil {
		p, err = layout.Write(cache, empty.Index)
		if err != nil {
			return p, fmt.Errorf("could not initialize cache at path %s: %v", cache, err)
		}
	}
	return p, nil
}
