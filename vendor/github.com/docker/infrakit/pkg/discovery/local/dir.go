package local

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"sync"

	"github.com/docker/infrakit/pkg/discovery"
	"github.com/docker/infrakit/pkg/plugin"
)

// Dir returns the directory to use for plugin discovery, which may be customized by the environment.
func Dir() string {
	if pluginDir := os.Getenv(discovery.PluginDirEnvVar); pluginDir != "" {
		return pluginDir
	}

	home := os.Getenv("HOME")
	if usr, err := user.Current(); err == nil {
		home = usr.HomeDir
	}
	return filepath.Join(home, ".infrakit/plugins")
}

// NewPluginDiscovery creates a plugin discovery based on the environment configuration.
func NewPluginDiscovery() (discovery.Plugins, error) {
	return NewPluginDiscoveryWithDirectory(Dir())
}

// NewPluginDiscoveryWithDirectory creates a plugin discovery based on the directory given.
func NewPluginDiscoveryWithDirectory(pluginDir string) (discovery.Plugins, error) {
	stat, err := os.Stat(pluginDir)
	if err == nil {
		if !stat.IsDir() {
			return nil, fmt.Errorf("Plugin dir %s is a file", pluginDir)
		}
	} else {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(pluginDir, 0700); err != nil {
				return nil, fmt.Errorf("Failed to create plugin dir %s: %s", pluginDir, err)
			}
		} else {
			return nil, fmt.Errorf("Failed to access plugin dir %s: %s", pluginDir, err)
		}
	}

	return newDirPluginDiscovery(pluginDir)
}

type dirPluginDiscovery struct {
	dir  string
	lock sync.Mutex
}

// Find returns a plugin by name
func (r *dirPluginDiscovery) Find(name plugin.Name) (*plugin.Endpoint, error) {
	lookup, _ := name.GetLookupAndType()
	plugins, err := r.List()
	if err != nil {
		return nil, err
	}

	p, exists := plugins[lookup]
	if !exists {
		return nil, discovery.ErrNotFound(string(name))
	}

	return p, nil
}

// newDirPluginDiscovery creates a registry instance with the given file directory path.
func newDirPluginDiscovery(dir string) (*dirPluginDiscovery, error) {
	d := &dirPluginDiscovery{dir: dir}

	// Perform a dummy read to catch obvious issues early (such as the directory not existing).
	_, err := d.List()
	return d, err
}

func (r *dirPluginDiscovery) dirLookup(entry os.FileInfo) (*plugin.Endpoint, error) {
	socketPath := filepath.Join(r.dir, entry.Name())
	if entry.Mode()&os.ModeSocket != 0 {
		return &plugin.Endpoint{
			Protocol: "unix",
			Address:  socketPath,
			Name:     entry.Name(),
		}, nil
	}

	return nil, discovery.ErrNotUnixSocket(socketPath)
}

// List returns a list of plugins known, keyed by the name
func (r *dirPluginDiscovery) List() (map[string]*plugin.Endpoint, error) {

	r.lock.Lock()
	defer r.lock.Unlock()

	entries, err := ioutil.ReadDir(r.dir)
	if err != nil {
		return nil, err
	}

	plugins := map[string]*plugin.Endpoint{}

	for _, entry := range entries {
		if !entry.IsDir() {

			instance, err := r.dirLookup(entry)

			if err != nil {
				if !discovery.IsErrNotUnixSocket(err) {
					log.Warn("Err loading plugin", "err", err)
				}
				continue
			}

			if instance == nil {
				log.Warn("Plugin is nil")
				continue
			}

			log.Debug("Discovered plugin", "address", instance.Address)
			plugins[instance.Name] = instance
		}
	}

	return plugins, nil
}
