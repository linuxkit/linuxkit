package docker

import (
	"context"
	"fmt"
	"io"

	"github.com/containerd/containerd/v2/pkg/reference"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/dockercontext"
	containertypes "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	log "github.com/sirupsen/logrus"
)

// Client creates a client for an explicitly resolved Docker context. Callers own
// the client and must close it after reading any response streams.
func Client(dockerContext string) (client.APIClient, error) {
	selection, err := dockercontext.Resolve(dockerContext)
	if err != nil {
		return nil, err
	}
	return selection.Client()
}

// HasImage check if the provided ref is available in the docker cache.
func HasImage(ref *reference.Spec, architecture, dockerContext string) error {
	log.Debugf("docker inspect image: %s", ref)
	cli, err := Client(dockerContext)
	if err != nil {
		return err
	}
	defer func() { _ = cli.Close() }()
	imageInspect, err := InspectImage(cli, ref)
	if err != nil {
		return fmt.Errorf("docker context %q: inspect image %s: %w", dockerContext, ref, err)
	}
	if imageInspect.Architecture != "" && imageInspect.Architecture != architecture {
		return fmt.Errorf("image not found for right architecture (%s != %s)", imageInspect.Architecture, architecture)
	}

	return err
}

// InspectImage inspect the provided ref.
func InspectImage(cli client.APIClient, ref *reference.Spec) (client.ImageInspectResult, error) {
	log.Debugf("docker inspect image: %s", ref)

	inspect, err := cli.ImageInspect(context.Background(), ref.String())
	if err != nil {
		return client.ImageInspectResult{}, err
	}

	log.Debugf("docker inspect image: %s...Done", ref)

	return inspect, nil
}

// Create create a container from the given image in docker, returning the full hash ID
// of the created container. Does not start the container.
func Create(image string, withNetwork bool, dockerContext string) (string, error) {
	log.Debugf("docker create: %s", image)
	cli, err := Client(dockerContext)
	if err != nil {
		return "", fmt.Errorf("could not initialize Docker API client: %w", err)
	}
	defer func() { _ = cli.Close() }()
	// we do not ever run the container, so /dev/null is used as command
	config := &containertypes.Config{
		Cmd:             []string{"/dev/null"},
		Image:           image,
		NetworkDisabled: !withNetwork,
	}

	respBody, err := cli.ContainerCreate(context.Background(), client.ContainerCreateOptions{Config: config})
	if err != nil {
		return "", fmt.Errorf("docker context %q: create container from image %q: %w", dockerContext, image, err)
	}

	log.Debugf("docker create: %s...Done", image)
	return respBody.ID, nil
}

// Export export the provided container ID from docker using `docker export`.
// The container must already exist.
func Export(container, dockerContext string) (io.ReadCloser, error) {
	log.Debugf("docker export: %s", container)
	cli, err := Client(dockerContext)
	if err != nil {
		return nil, fmt.Errorf("could not initialize Docker API client: %w", err)
	}
	r, err := cli.ContainerExport(context.Background(), container, client.ContainerExportOptions{})
	if err != nil {
		_ = cli.Close()
		return nil, fmt.Errorf("docker context %q: export container %q: %w", dockerContext, container, err)
	}
	return readCloser{r: r, closer: func() error { defer func() { _ = cli.Close() }(); return r.Close() }}, nil
}

// Save save the provided image ref.
func Save(image, dockerContext string) (io.ReadCloser, error) {
	log.Debugf("docker save: %s", image)
	cli, err := Client(dockerContext)
	if err != nil {
		return nil, fmt.Errorf("could not initialize Docker API client: %w", err)
	}
	r, err := cli.ImageSave(context.Background(), []string{image})
	if err != nil {
		_ = cli.Close()
		return nil, fmt.Errorf("docker context %q: save image %q: %w", dockerContext, image, err)
	}
	return readCloser{r: r, closer: func() error { defer func() { _ = cli.Close() }(); return r.Close() }}, nil
}

// Rm remove the given container from docker.
func Rm(container, dockerContext string) error {
	log.Debugf("docker rm: %s", container)
	cli, err := Client(dockerContext)
	if err != nil {
		return fmt.Errorf("could not initialize Docker API client: %w", err)
	}
	defer func() { _ = cli.Close() }()
	if _, err = cli.ContainerRemove(context.Background(), container, client.ContainerRemoveOptions{}); err != nil {
		return fmt.Errorf("docker context %q: remove container %q: %w", dockerContext, container, err)
	}
	log.Debugf("docker rm: %s...Done", container)
	return nil
}
