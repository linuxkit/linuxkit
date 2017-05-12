package main

// We want to replace much of this with use of containerd tools
// and also using the Docker API not shelling out

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
)

func dockerRun(args ...string) ([]byte, error) {
	log.Debugf("docker run: %s", strings.Join(args, " "))
	docker, err := exec.LookPath("docker")
	if err != nil {
		return []byte{}, errors.New("Docker does not seem to be installed")
	}
	args = append([]string{"run", "--rm", "--log-driver=none"}, args...)
	cmd := exec.Command(docker, args...)

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return []byte{}, err
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return []byte{}, err
	}

	err = cmd.Start()
	if err != nil {
		return []byte{}, err
	}

	stdout, err := ioutil.ReadAll(stdoutPipe)
	if err != nil {
		return []byte{}, err
	}

	stderr, err := ioutil.ReadAll(stderrPipe)
	if err != nil {
		return []byte{}, err
	}

	err = cmd.Wait()
	if err != nil {
		return []byte{}, fmt.Errorf("%v: %s", err, stderr)
	}

	log.Debugf("docker run: %s...Done", strings.Join(args, " "))
	return stdout, nil
}

func dockerRunInput(input io.Reader, args ...string) ([]byte, error) {
	log.Debugf("docker run (input): %s", strings.Join(args, " "))
	docker, err := exec.LookPath("docker")
	if err != nil {
		return []byte{}, errors.New("Docker does not seem to be installed")
	}
	args = append([]string{"run", "--rm", "-i"}, args...)
	cmd := exec.Command(docker, args...)
	cmd.Stdin = input

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return []byte{}, err
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return []byte{}, err
	}

	err = cmd.Start()
	if err != nil {
		return []byte{}, err
	}

	stdout, err := ioutil.ReadAll(stdoutPipe)
	if err != nil {
		return []byte{}, err
	}

	stderr, err := ioutil.ReadAll(stderrPipe)
	if err != nil {
		return []byte{}, err
	}

	err = cmd.Wait()
	if err != nil {
		return []byte{}, fmt.Errorf("%v: %s", err, stderr)
	}

	log.Debugf("docker run (input): %s...Done", strings.Join(args, " "))
	return stdout, nil
}

func dockerCreate(image string) (string, error) {
	log.Debugf("docker create: %s", image)
	cli, err := dockerClient()
	if err != nil {
		return "", errors.New("could not initialize Docker API client")
	}
	// we do not ever run the container, so /dev/null is used as command
	config := &container.Config{
		Cmd:   []string{"/dev/null"},
		Image: image,
	}
	respBody, err := cli.ContainerCreate(context.Background(), config, nil, nil, "")
	if err != nil {
		return "", err
	}

	log.Debugf("docker create: %s...Done", image)
	return respBody.ID, nil
}

func dockerExport(container string) ([]byte, error) {
	log.Debugf("docker export: %s", container)
	cli, err := dockerClient()
	if err != nil {
		return []byte{}, errors.New("could not initialize Docker API client")
	}
	responseBody, err := cli.ContainerExport(context.Background(), container)
	if err != nil {
		return []byte{}, err
	}
	defer responseBody.Close()

	output := bytes.NewBuffer(nil)
	_, err = io.Copy(output, responseBody)
	if err != nil {
		return []byte{}, err
	}

	return output.Bytes(), nil
}

func dockerRm(container string) error {
	log.Debugf("docker rm: %s", container)
	cli, err := dockerClient()
	if err != nil {
		return errors.New("could not initialize Docker API client")
	}
	if err = cli.ContainerRemove(context.Background(), container, types.ContainerRemoveOptions{}); err != nil {
		return err
	}
	log.Debugf("docker rm: %s...Done", container)
	return nil
}

func dockerPull(image string, trustedPull bool) error {
	log.Debugf("docker pull: %s", image)
	if trustedPull {
		log.Debugf("pulling %s with content trust", image)
		trustedImg, err := TrustedReference(image)
		if err != nil {
			return fmt.Errorf("Trusted pull for %s failed: %v", image, err)
		}
		image = trustedImg.String()
	}
	cli, err := dockerClient()
	if err != nil {
		return errors.New("could not initialize Docker API client")
	}

	r, err := cli.ImagePull(context.Background(), image, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer r.Close()
	_, err = io.Copy(ioutil.Discard, r)
	if err != nil {
		return err
	}
	log.Debugf("docker pull: %s...Done", image)
	return nil
}

func dockerClient() (*client.Client, error) {
	// for maximum compatibility as we use nothing new
	err := os.Setenv("DOCKER_API_VERSION", "1.23")
	if err != nil {
		return nil, err
	}
	return client.NewEnvClient()
}

func dockerInspectImage(cli *client.Client, image string) (types.ImageInspect, error) {
	log.Debugf("docker inspect image: %s", image)

	inspect, _, err := cli.ImageInspectWithRaw(context.Background(), image)
	if err != nil {
		if client.IsErrImageNotFound(err) {
			pullErr := dockerPull(image, false)
			if pullErr != nil {
				return types.ImageInspect{}, pullErr
			}
			inspect, _, err = cli.ImageInspectWithRaw(context.Background(), image)
			if err != nil {
				return types.ImageInspect{}, err
			}
		} else {
			return types.ImageInspect{}, err
		}
	}

	log.Debugf("docker inspect image: %s...Done", image)

	return inspect, nil
}
