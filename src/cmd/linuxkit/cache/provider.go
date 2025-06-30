package cache

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/containerd/containerd/v2/core/content"
	"github.com/containerd/containerd/v2/plugins/content/local"
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
