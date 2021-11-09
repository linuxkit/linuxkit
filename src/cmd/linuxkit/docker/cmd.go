package docker

import (
	"context"
	"errors"
	"io"
	"os"

	"github.com/containerd/containerd/reference"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
)

// Client get a docker client.
func Client() (*client.Client, error) {
	// for maximum compatibility as we use nothing new
	// 1.30 corresponds to Docker 17.06, supported until 2020.
	err := os.Setenv("DOCKER_API_VERSION", "1.30")
	if err != nil {
		return nil, err
	}
	return client.NewEnvClient()
}

// HasImage check if the provided ref is available in the docker cache.
func HasImage(ref *reference.Spec) error {
	log.Debugf("docker inspect image: %s", ref)
	cli, err := Client()
	if err != nil {
		return err
	}
	_, err = InspectImage(cli, ref)

	return err
}

// InspectImage inspect the provided ref.
func InspectImage(cli *client.Client, ref *reference.Spec) (dockertypes.ImageInspect, error) {
	log.Debugf("docker inspect image: %s", ref)

	inspect, _, err := cli.ImageInspectWithRaw(context.Background(), ref.String())
	if err != nil {
		return dockertypes.ImageInspect{}, err
	}

	log.Debugf("docker inspect image: %s...Done", ref)

	return inspect, nil
}

// Create create a container from the given image in docker, returning the full hash ID
// of the created container. Does not start the container.
func Create(image string) (string, error) {
	log.Debugf("docker create: %s", image)
	cli, err := Client()
	if err != nil {
		return "", errors.New("could not initialize Docker API client")
	}
	// we do not ever run the container, so /dev/null is used as command
	config := &container.Config{
		Cmd:   []string{"/dev/null"},
		Image: image,
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
	responseBody, err := cli.ContainerExport(context.Background(), container)
	if err != nil {
		return nil, err
	}

	return responseBody, err
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
	if err = cli.ContainerRemove(context.Background(), container, dockertypes.ContainerRemoveOptions{}); err != nil {
		return err
	}
	log.Debugf("docker rm: %s...Done", container)
	return nil
}
