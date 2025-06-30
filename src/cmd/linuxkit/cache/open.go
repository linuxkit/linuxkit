package cache

import (
	"fmt"
	"os"

	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/layout"
)

// Get get or initialize the cache
func (p *Provider) Get(cache string) (layout.Path, error) {
	// ensure the dir exists
	if err := os.MkdirAll(cache, os.ModePerm); err != nil {
		return "", fmt.Errorf("unable to create cache directory %s: %v", cache, err)
	}

	// first try to read the layout from the path
	// if it exists, we can use it
	// if it does not exist, we will initialize it
	//
	// do not lock for first read, because we do not need the lock except for initialization
	// and future writes, so why slow down reads?
	l, err := layout.FromPath(cache)

	// initialize the cache path if needed
	if err != nil {
		if err := p.Lock(); err != nil {
			return "", fmt.Errorf("unable to lock cache %s: %v", cache, err)
		}
		defer p.Unlock()

		// after lock, try to read the layout again
		// in case another process initialized it while we were waiting for the lock
		// if it still does not exist, we will initialize it
		l, err = layout.FromPath(cache)
		if err != nil {
			l, err = layout.Write(cache, empty.Index)
			if err != nil {
				return l, fmt.Errorf("could not initialize cache at path %s: %v", cache, err)
			}
		}
	}
	return l, nil
}
