package cli

import (
	log "github.com/Sirupsen/logrus"
	"github.com/docker/infrakit/discovery"
	"github.com/docker/infrakit/plugin/util/server"
	"net/http"
	"path"
)

// RunPlugin runs a plugin server, advertising with the provided name for discovery.
func RunPlugin(name string, plugin http.Handler) {
	_, stopped, err := server.StartPluginAtPath(path.Join(discovery.Dir(), name), plugin)
	if err != nil {
		log.Error(err)
	}

	<-stopped // block until done
}
