package discovery

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/infrakit/plugin"
	"github.com/docker/infrakit/plugin/util/client"
)

type pluginInstance struct {
	name     string
	endpoint string
	client   *client.Client
}

// String returns a string representation of the callable.
func (i *pluginInstance) String() string {
	return i.endpoint
}

// Call calls the plugin with some message
func (i *pluginInstance) Call(endpoint plugin.Endpoint, message, result interface{}) ([]byte, error) {
	return i.client.Call(endpoint, message, result)
}

type dirPluginDiscovery struct {
	dir  string
	lock sync.Mutex
}

// Find returns a plugin by name
func (r *dirPluginDiscovery) Find(name string) (plugin.Callable, error) {

	plugins, err := r.List()
	if err != nil {
		return nil, err
	}

	p, exists := plugins[name]
	if !exists {
		return nil, fmt.Errorf("Plugin not found: %s", name)
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

func (r *dirPluginDiscovery) dirLookup(entry os.FileInfo) (*pluginInstance, error) {
	if entry.Mode()&os.ModeSocket != 0 {
		socketPath := filepath.Join(r.dir, entry.Name())
		return &pluginInstance{
			endpoint: socketPath,
			name:     entry.Name(),
			client:   client.New(socketPath),
		}, nil
	}

	return nil, fmt.Errorf("File is not a socket: %s", entry)
}

// List returns a list of plugins known, keyed by the name
func (r *dirPluginDiscovery) List() (map[string]plugin.Callable, error) {

	r.lock.Lock()
	defer r.lock.Unlock()

	log.Debugln("Opening:", r.dir)
	entries, err := ioutil.ReadDir(r.dir)
	if err != nil {
		return nil, err
	}

	plugins := map[string]plugin.Callable{}

	for _, entry := range entries {
		if !entry.IsDir() {

			instance, err := r.dirLookup(entry)
			if err != nil || instance == nil {
				log.Warningln("Loading plugin err=", err)
				continue
			}

			log.Debugln("Discovered plugin at", instance.endpoint)
			plugins[instance.name] = plugin.Callable(instance)
		}
	}

	return plugins, nil
}
