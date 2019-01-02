package moby

// We want to replace much of this with use of containerd tools
// and also using the Docker API not shelling out

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/containerd/containerd/reference"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	distref "github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/docker/registry"
	log "github.com/sirupsen/logrus"
)

var (
	// configFile is a cached copy of the client config file
	configFile *configfile.ConfigFile
	// authCache maps a registry URL to encoded authentication
	authCache map[string]string
)

func dockerRun(input io.Reader, output io.Writer, trust bool, img string, args ...string) error {
	log.Debugf("docker run %s (trust=%t) (input): %s", img, trust, strings.Join(args, " "))
	docker, err := exec.LookPath("docker")
	if err != nil {
		return errors.New("Docker does not seem to be installed")
	}

	env := os.Environ()
	if trust {
		env = append(env, "DOCKER_CONTENT_TRUST=1")
	}

	// Pull first to avoid https://github.com/docker/cli/issues/631
	pull := exec.Command(docker, "pull", img)
	pull.Env = env
	if err := pull.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("docker pull %s failed: %v output:\n%s", img, err, exitError.Stderr)
		}
		return err
	}

	args = append([]string{"run", "--network=none", "--rm", "-i", img}, args...)
	cmd := exec.Command(docker, args...)
	cmd.Stdin = input
	cmd.Stdout = output
	cmd.Env = env

	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("docker run %s failed: %v output:\n%s", img, err, exitError.Stderr)
		}
		return err
	}

	log.Debugf("docker run %s (input): %s...Done", img, strings.Join(args, " "))
	return nil
}

func dockerCreate(image string) (string, error) {
	log.Debugf("docker create: %s", image)
	cli, err := dockerClient()
	if err != nil {
		return "", errors.New("could not initialize Docker API client: " + err.Error())
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

func dockerExport(container string) (io.ReadCloser, error) {
	log.Debugf("docker export: %s", container)
	cli, err := dockerClient()
	if err != nil {
		return nil, errors.New("could not initialize Docker API client: " + err.Error())
	}
	responseBody, err := cli.ContainerExport(context.Background(), container)
	if err != nil {
		return nil, err
	}

	return responseBody, err
}

func dockerRm(container string) error {
	log.Debugf("docker rm: %s", container)
	cli, err := dockerClient()
	if err != nil {
		return errors.New("could not initialize Docker API client: " + err.Error())
	}
	if err = cli.ContainerRemove(context.Background(), container, types.ContainerRemoveOptions{}); err != nil {
		return err
	}
	log.Debugf("docker rm: %s...Done", container)
	return nil
}

func dockerPull(ref *reference.Spec, forcePull, trustedPull bool) error {
	log.Debugf("docker pull: %s", ref)
	cli, err := dockerClient()
	if err != nil {
		return errors.New("could not initialize Docker API client: " + err.Error())
	}

	if trustedPull {
		log.Debugf("pulling %s with content trust", ref)
		trustedImg, err := TrustedReference(ref.String())
		if err != nil {
			return fmt.Errorf("Trusted pull for %s failed: %v", ref, err)
		}

		// tag the image on a best-effort basis after pulling with content trust,
		// ensuring that docker picks up the tag and digest fom the canonical format
		defer func(src, dst string) {
			if err := cli.ImageTag(context.Background(), src, dst); err != nil {
				log.Debugf("could not tag trusted image %s to %s", src, dst)
			}
		}(trustedImg.String(), ref.String())

		log.Debugf("successfully verified trusted reference %s from notary", trustedImg.String())
		trustedSpec, err := reference.Parse(trustedImg.String())
		if err != nil {
			return fmt.Errorf("failed to convert trusted img %s to Spec: %v", trustedImg, err)
		}
		ref.Locator = trustedSpec.Locator
		ref.Object = trustedSpec.Object

		imageSearchArg := filters.NewArgs()
		imageSearchArg.Add("reference", trustedImg.String())
		if _, err := cli.ImageList(context.Background(), types.ImageListOptions{Filters: imageSearchArg}); err == nil && !forcePull {
			log.Debugf("docker pull: trusted image %s already cached...Done", trustedImg.String())
			return nil
		}
	}

	log.Infof("Pull image: %s", ref)
	pullOptions := types.ImagePullOptions{RegistryAuth: getAuth(cli, ref.String())}
	r, err := cli.ImagePull(context.Background(), ref.String(), pullOptions)
	if err != nil {
		return err
	}
	defer r.Close()
	_, err = io.Copy(ioutil.Discard, r)
	if err != nil {
		return err
	}
	log.Debugf("docker pull: %s...Done", ref)
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

func dockerInspectImage(cli *client.Client, ref *reference.Spec, trustedPull bool) (types.ImageInspect, error) {
	log.Debugf("docker inspect image: %s", ref)

	inspect, _, err := cli.ImageInspectWithRaw(context.Background(), ref.String())
	if err != nil {
		if client.IsErrNotFound(err) {
			pullErr := dockerPull(ref, true, trustedPull)
			if pullErr != nil {
				return types.ImageInspect{}, pullErr
			}
			inspect, _, err = cli.ImageInspectWithRaw(context.Background(), ref.String())
			if err != nil {
				return types.ImageInspect{}, err
			}
		} else {
			return types.ImageInspect{}, err
		}
	}

	log.Debugf("docker inspect image: %s...Done", ref)

	return inspect, nil
}

// getAuth attempts to get an encoded authentication spec for a reference
// This function does not error. It simply returns a empty string if something fails.
func getAuth(cli *client.Client, refStr string) string {
	// Initialise state on first use
	if authCache == nil {
		authCache = make(map[string]string)
	}
	if configFile == nil {
		configFile = config.LoadDefaultConfigFile(ioutil.Discard)
	}

	// Normalise ref
	ref, err := distref.ParseNormalizedNamed(refStr)
	if err != nil {
		log.Debugf("Failed to parse reference %s: %v: %v", refStr, err)
		return ""
	}
	hostname := distref.Domain(ref)
	// Special case for docker.io which currently is stored as https://index.docker.io/v1/ in authentication map.
	// Other registries are stored under their domain name.
	if hostname == registry.IndexName {
		hostname = registry.IndexServer
	}

	// Look up in cache
	if val, ok := authCache[hostname]; ok {
		log.Debugf("Returning cached credentials for %s", hostname)
		return val
	}

	authConfig, err := configFile.GetAuthConfig(hostname)
	if err != nil {
		log.Debugf("Failed to get authentication config for %s. Ignoring: %v", hostname, err)
		return ""
	}
	log.Debugf("Looked up credentials for %s", hostname)

	encodedAuth, err := command.EncodeAuthToBase64(authConfig)
	if err != nil {
		return ""
	}
	authCache[hostname] = encodedAuth
	return encodedAuth
}
