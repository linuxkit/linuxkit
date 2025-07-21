package cache

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/containerd/containerd/v2/core/content"
	"github.com/containerd/containerd/v2/plugins/content/local"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/util"
	log "github.com/sirupsen/logrus"
)

// Provider cache implementation of cacheProvider
type Provider struct {
	cache   layout.Path
	store   content.Store
	dir     string
	lock    *util.FileLock
	lockMut sync.Mutex
}

// NewProvider create a new CacheProvider based in the provided directory
func NewProvider(dir string) (*Provider, error) {
	p := &Provider{dir: dir, lockMut: sync.Mutex{}}
	layout, err := p.Get(dir)
	if err != nil {
		return nil, err
	}
	store, err := local.NewStore(dir)
	if err != nil {
		return nil, err
	}
	p.cache = layout
	p.store = store
	return p, nil
}

// Index returns the root image index for the cache.
// All attempts to read the index *must* use this function, so that it will lock the cache to prevent concurrent access.
// The underlying library writes modifications directly to the index,
// so not only must ensure that that only one process is writing at a time, but that no one is reading
// while we are writing, to avoid corruption.
func (p *Provider) Index() (v1.ImageIndex, error) {
	if p.Lock() != nil {
		return nil, fmt.Errorf("unable to lock cache %s", p.dir)
	}
	defer p.Unlock()
	return p.cache.ImageIndex()
}

// Lock locks the cache directory to prevent concurrent access
func (p *Provider) Lock() error {
	// if the lock is already set, we do not need to do anything
	if p.lock != nil {
		return nil
	}
	p.lockMut.Lock()
	defer p.lockMut.Unlock()
	var lockFile = filepath.Join(p.dir, lockfile)
	lock, err := util.Lock(lockFile)
	if err != nil {
		return fmt.Errorf("unable to retrieve cache lock %s: %v", lockFile, err)
	}
	p.lock = lock
	return nil
}

// Unlock releases the lock on the cache directory
func (p *Provider) Unlock() {
	p.lockMut.Lock()
	defer p.lockMut.Unlock()
	// if the lock is not set, we do not need to do anything
	if p.lock == nil {
		return
	}
	var lockFile = filepath.Join(p.dir, lockfile)
	if err := p.lock.Unlock(); err != nil {
		log.Errorf("unable to close lock for cache %s: %v", lockFile, err)
	}
	p.lock = nil
}
