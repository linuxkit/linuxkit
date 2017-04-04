package cli

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/Sirupsen/logrus"
	"github.com/docker/infrakit/pkg/discovery/local"
	"github.com/docker/infrakit/pkg/rpc/server"
)

// EnsureDirExists makes sure the directory where the socket file will be placed exists.
func EnsureDirExists(dir string) {
	os.MkdirAll(dir, 0700)
}

// RunPlugin runs a plugin server, advertising with the provided name for discovery.
// The plugin should conform to the rpc call convention as implemented in the rpc package.
func RunPlugin(name string, plugin server.VersionedInterface, more ...server.VersionedInterface) {

	dir := local.Dir()
	EnsureDirExists(dir)

	socketPath := path.Join(dir, name)
	pidPath := path.Join(dir, name+".pid")

	stoppable, err := server.StartPluginAtPath(socketPath, plugin, more...)
	if err != nil {
		logrus.Error(err)
	}

	// write PID file
	err = ioutil.WriteFile(pidPath, []byte(fmt.Sprintf("%v", os.Getpid())), 0644)
	if err != nil {
		logrus.Error(err)
	}
	logrus.Infoln("PID file at", pidPath)
	if stoppable != nil {
		stoppable.AwaitStopped()
	}

	// clean up
	os.Remove(pidPath)
	logrus.Infoln("Removed PID file at", pidPath)
}
