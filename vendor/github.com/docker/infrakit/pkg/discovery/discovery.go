package discovery

import (
	"fmt"
	"os"
	"os/user"
	"path"

	"github.com/docker/infrakit/pkg/plugin"
)

// Plugins provides access to plugin discovery.
type Plugins interface {
	// Find looks up the plugin by name.  The name can be of the form $lookup[/$subtype].  See GetLookupAndType().
	Find(name plugin.Name) (*plugin.Endpoint, error)
	List() (map[string]*plugin.Endpoint, error)
}

const (
	// PluginDirEnvVar is the environment variable that may be used to customize the plugin discovery path.
	PluginDirEnvVar = "INFRAKIT_PLUGINS_DIR"
)

// Dir returns the directory to use for plugin discovery, which may be customized by the environment.
func Dir() string {
	if pluginDir := os.Getenv(PluginDirEnvVar); pluginDir != "" {
		return pluginDir
	}

	home := os.Getenv("HOME")
	if usr, err := user.Current(); err == nil {
		home = usr.HomeDir
	}
	return path.Join(home, ".infrakit/plugins")
}

// NewPluginDiscovery creates a plugin discovery based on the environment configuration.
func NewPluginDiscovery() (Plugins, error) {
	return NewPluginDiscoveryWithDirectory(Dir())
}

// NewPluginDiscoveryWithDirectory creates a plugin discovery based on the directory given.
func NewPluginDiscoveryWithDirectory(pluginDir string) (Plugins, error) {
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
