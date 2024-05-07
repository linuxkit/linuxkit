package build

// We want to replace much of this with use of containerd tools
// and also using the Docker API not shelling out

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
)

// dockerRun is outside the linuxkit/docker package, because that is for caching, this is
// used for running to build images. runEnv is passed through to the docker run command.
func dockerRun(input io.Reader, output io.Writer, img string, runEnv []string, imageArgs ...string) error {
	log.Debugf("docker run %s (input): %s", img, strings.Join(imageArgs, " "))
	docker, err := exec.LookPath("docker")
	if err != nil {
		return errors.New("docker does not seem to be installed")
	}

	env := os.Environ()

	// Pull first to avoid https://github.com/docker/cli/issues/631
	pull := exec.Command(docker, "pull", img)
	pull.Env = env
	if err := pull.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("docker pull %s failed: %v output:\n%s", img, err, exitError.Stderr)
		}
		return err
	}

	var errbuf strings.Builder
	args := []string{"run", "--network=none", "--log-driver=none", "--rm", "-i"}
	for _, e := range runEnv {
		args = append(args, "-e", e)
	}

	args = append(args, img)
	args = append(args, imageArgs...)
	cmd := exec.Command(docker, args...)
	cmd.Stderr = &errbuf
	cmd.Stdin = input
	cmd.Stdout = output
	cmd.Env = env

	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("docker run %s failed: %v output:\n%s", img, err, errbuf.String())
		}
		return err
	}

	log.Debugf("docker run %s (input): %s...Done", img, strings.Join(args, " "))
	return nil
}
