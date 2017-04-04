package local

import (
	"fmt"

	"github.com/docker/infrakit/pkg/discovery"
	logutil "github.com/docker/infrakit/pkg/log"
	"github.com/spf13/afero"
)

// Setup sets up the necessary environment for running this module -- ie make sure
// the CLI module directories are present, etc.
func Setup() error {
	dir := Dir()
	if dir == "" {
		return fmt.Errorf("Env not set:%s", discovery.PluginDirEnvVar)
	}
	fs := afero.NewOsFs()
	exists, err := afero.Exists(fs, dir)
	if err != nil {
		return err
	}
	if !exists {
		log.Warn("Creating directory", "dir", dir)
		err = fs.MkdirAll(dir, 0600)
		if err != nil {
			return err
		}
	}
	return nil
}

var log = logutil.New("module", "discovery/local")
