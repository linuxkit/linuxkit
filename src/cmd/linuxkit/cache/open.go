package cache

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/util"
	log "github.com/sirupsen/logrus"
)

var (
	newIndexLockFile = filepath.Join(os.TempDir(), "linuxkit-new-cache-index.lock")
)

// Get get or initialize the cache
func Get(cache string) (layout.Path, error) {
	// initialize the cache path if needed
	p, err := layout.FromPath(cache)
	if err != nil {
		if err := os.WriteFile(newIndexLockFile, []byte{}, 0644); err != nil {
			return "", fmt.Errorf("unable to create lock file %s for writing descriptor for new cache %s: %v", newIndexLockFile, cache, err)
		}
		lock, err := util.Lock(newIndexLockFile)
		if err != nil {
			return "", fmt.Errorf("unable to retrieve lock for writing descriptor for new cache %s: %v", newIndexLockFile, err)
		}
		defer func() {
			if err := lock.Unlock(); err != nil {
				log.Errorf("unable to close lock for cache index after writing descriptor for new cache: %v", err)
			}
			if err := os.RemoveAll(newIndexLockFile); err != nil {
				log.Errorf("unable to remove lock file %s after writing descriptor for new cache: %v", newIndexLockFile, err)
			}
		}()
		p, err = layout.Write(cache, empty.Index)
		if err != nil {
			return p, fmt.Errorf("could not initialize cache at path %s: %v", cache, err)
		}
	}
	return p, nil
}
