package registry

import (
	"os"

	"github.com/docker/cli/cli/config"
	dockertypes "github.com/docker/docker/api/types"
)

const (
	registryServer = "https://index.docker.io/v1/"
)

// GetDockerAuth get an AuthConfig for the default registry server.
func GetDockerAuth() (dockertypes.AuthConfig, error) {
	cfgFile := config.LoadDefaultConfigFile(os.Stderr)
	authconfig, err := cfgFile.GetAuthConfig(registryServer)
	return dockertypes.AuthConfig(authconfig), err
}
