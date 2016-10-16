package discovery

import (
	"fmt"
	"github.com/docker/infrakit/plugin"
	"os"
	"os/user"
	"path"
)

// Plugins provides access to plugin discovery.
type Plugins interface {
	Find(name string) (plugin.Callable, error)

	List() (map[string]plugin.Callable, error)
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

	usr, err := user.Current()
	if err != nil {
		panic(err)
	}
	return path.Join(usr.HomeDir, ".infrakit/plugins")
}

// NewPluginDiscovery creates a plugin discovery based on the environment configuration.
func NewPluginDiscovery() (Plugins, error) {
	pluginDir := Dir()

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
