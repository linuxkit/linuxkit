package discovery

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/infrakit/pkg/plugin"
)

type errNotUnixSocket string

func (e errNotUnixSocket) Error() string {
	return string(e)
}

// IsErrNotUnixSocket returns true if the error is due to the file not being a valid unix socket.
func IsErrNotUnixSocket(e error) bool {
	_, is := e.(errNotUnixSocket)
	return is
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
		return nil, fmt.Errorf("Plugin not found: %s (looked up using %s)", name, lookup)
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
	if entry.Mode()&os.ModeSocket != 0 {
		socketPath := filepath.Join(r.dir, entry.Name())
		return &plugin.Endpoint{
			Protocol: "unix",
			Address:  socketPath,
			Name:     entry.Name(),
		}, nil
	}

	return nil, errNotUnixSocket(fmt.Sprintf("File is not a socket: %s", entry))
}

// List returns a list of plugins known, keyed by the name
func (r *dirPluginDiscovery) List() (map[string]*plugin.Endpoint, error) {

	r.lock.Lock()
	defer r.lock.Unlock()

	log.Debugln("Opening:", r.dir)
	entries, err := ioutil.ReadDir(r.dir)
	if err != nil {
		return nil, err
	}

	plugins := map[string]*plugin.Endpoint{}

	for _, entry := range entries {
		if !entry.IsDir() {

			instance, err := r.dirLookup(entry)

			if err != nil {
				if !IsErrNotUnixSocket(err) {
					log.Warningln("Loading plugin err=", err)
				}
				continue
			}

			if instance == nil {
				log.Warningln("Plugin in nil=")
				continue
			}

			log.Debugln("Discovered plugin at", instance.Address)
			plugins[instance.Name] = instance
		}
	}

	return plugins, nil
}
