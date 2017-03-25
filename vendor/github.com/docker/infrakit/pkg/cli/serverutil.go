package cli

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/infrakit/pkg/discovery"
	"github.com/docker/infrakit/pkg/rpc/server"
)

// EnsureDirExists makes sure the directory where the socket file will be placed exists.
func EnsureDirExists(dir string) {
	os.MkdirAll(dir, 0700)
}

// RunPlugin runs a plugin server, advertising with the provided name for discovery.
// The plugin should conform to the rpc call convention as implemented in the rpc package.
func RunPlugin(name string, plugin server.VersionedInterface, more ...server.VersionedInterface) {

	dir := discovery.Dir()
	EnsureDirExists(dir)

	socketPath := path.Join(dir, name)
	pidPath := path.Join(dir, name+".pid")

	stoppable, err := server.StartPluginAtPath(socketPath, plugin, more...)
	if err != nil {
		log.Error(err)
	}

	// write PID file
	err = ioutil.WriteFile(pidPath, []byte(fmt.Sprintf("%v", os.Getpid())), 0644)
	if err != nil {
		log.Error(err)
	}
	log.Infoln("PID file at", pidPath)
	if stoppable != nil {
		stoppable.AwaitStopped()
	}

	// clean up
	os.Remove(pidPath)
	log.Infoln("Removed PID file at", pidPath)
}
