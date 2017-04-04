package discovery

import (
	"fmt"

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

// ErrNotUnixSocket is the error raised when the file is not a unix socket
type ErrNotUnixSocket string

func (e ErrNotUnixSocket) Error() string {
	return fmt.Sprintf("not a unix socket:%s", string(e))
}

// IsErrNotUnixSocket returns true if the error is due to the file not being a valid unix socket.
func IsErrNotUnixSocket(e error) bool {
	_, is := e.(ErrNotUnixSocket)
	return is
}

// ErrNotFound is the error raised when the plugin is not found
type ErrNotFound string

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("plugin not found:%s", string(e))
}

// IsErrNotFound returns true if the error is due to a plugin not found.
func IsErrNotFound(e error) bool {
	_, is := e.(ErrNotFound)
	return is
}
