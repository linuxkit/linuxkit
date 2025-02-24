package docker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/containerd/containerd/v2/pkg/reference"
	"github.com/docker/cli/cli/connhelper"
	containertypes "github.com/docker/docker/api/types/container"
	dockerimagetypes "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
)

var (
	clientOnce     sync.Once
	memoizedClient *client.Client
	errClient      error
)

// Client get a docker client.
func Client() (*client.Client, error) {
	clientOnce.Do(func() {
		memoizedClient, errClient = createClient()
	})
	return memoizedClient, errClient
}

func createClient() (*client.Client, error) {
	options := []client.Opt{
		client.WithAPIVersionNegotiation(),
		client.WithTLSClientConfigFromEnv(),
		client.WithHostFromEnv(),
	}

	// Support connection over ssh.
	if host := os.Getenv(client.EnvOverrideHost); host != "" {
		helper, err := connhelper.GetConnectionHelper(host)
		if err != nil {
			return nil, err
		}
		if helper != nil {
			options = append(options, client.WithDialContext(helper.Dialer))
		}
	}

	return client.NewClientWithOpts(options...)
}

// HasImage check if the provided ref is available in the docker cache.
func HasImage(ref *reference.Spec, architecture string) error {
	log.Debugf("docker inspect image: %s", ref)
	cli, err := Client()
	if err != nil {
		return err
	}
	imageInspect, err := InspectImage(cli, ref)
	if err != nil {
		return err
	}
	if imageInspect.Architecture != "" && imageInspect.Architecture != architecture {
		return fmt.Errorf("image not found for right architecture (%s != %s)", imageInspect.Architecture, architecture)
	}

	return err
}

// InspectImage inspect the provided ref.
func InspectImage(cli *client.Client, ref *reference.Spec) (dockerimagetypes.InspectResponse, error) {
	log.Debugf("docker inspect image: %s", ref)

	inspect, err := cli.ImageInspect(context.Background(), ref.String())
	if err != nil {
		return dockerimagetypes.InspectResponse{}, err
	}

	log.Debugf("docker inspect image: %s...Done", ref)

	return inspect, nil
}

// Create create a container from the given image in docker, returning the full hash ID
// of the created container. Does not start the container.
func Create(image string, withNetwork bool) (string, error) {
	log.Debugf("docker create: %s", image)
	cli, err := Client()
	if err != nil {
		return "", errors.New("could not initialize Docker API client")
	}
	// we do not ever run the container, so /dev/null is used as command
	config := &containertypes.Config{
		Cmd:             []string{"/dev/null"},
		Image:           image,
		NetworkDisabled: !withNetwork,
	}

	respBody, err := cli.ContainerCreate(context.Background(), config, nil, nil, nil, "")
	if err != nil {
		return "", err
	}

	log.Debugf("docker create: %s...Done", image)
	return respBody.ID, nil
}

// Export export the provided container ID from docker using `docker export`.
// The container must already exist.
func Export(container string) (io.ReadCloser, error) {
	log.Debugf("docker export: %s", container)
	cli, err := Client()
	if err != nil {
		return nil, errors.New("could not initialize Docker API client")
	}
	return cli.ContainerExport(context.Background(), container)
}

// Save save the provided image ref.
func Save(image string) (io.ReadCloser, error) {
	log.Debugf("docker save: %s", image)
	cli, err := Client()
	if err != nil {
		return nil, errors.New("could not initialize Docker API client")
	}
	return cli.ImageSave(context.Background(), []string{image})
}

// Rm remove the given container from docker.
func Rm(container string) error {
	log.Debugf("docker rm: %s", container)
	cli, err := Client()
	if err != nil {
		return errors.New("could not initialize Docker API client")
	}
	if err = cli.ContainerRemove(context.Background(), container, containertypes.RemoveOptions{}); err != nil {
		return err
	}
	log.Debugf("docker rm: %s...Done", container)
	return nil
}
