package cache

import (
	"fmt"
	"path/filepath"

	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/util"
	log "github.com/sirupsen/logrus"
)

// Get get or initialize the cache
func Get(cache string) (layout.Path, error) {
	// initialize the cache path if needed
	p, err := layout.FromPath(cache)
	if err != nil {
		lock, err := util.Lock(filepath.Join(cache, indexFile))
		if err != nil {
			return "", fmt.Errorf("unable to lock cache index for writing descriptor for new cache: %v", err)
		}
		defer func() {
			if err := lock.Unlock(); err != nil {
				log.Errorf("unable to close lock for cache index after writing descriptor for new cache: %v", err)
			}
		}()
		p, err = layout.Write(cache, empty.Index)
		if err != nil {
			return p, fmt.Errorf("could not initialize cache at path %s: %v", cache, err)
		}
	}
	return p, nil
}
