package moby

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
// used for running to build images.
func dockerRun(input io.Reader, output io.Writer, img string, args ...string) error {
	log.Debugf("docker run %s (input): %s", img, strings.Join(args, " "))
	docker, err := exec.LookPath("docker")
	if err != nil {
		return errors.New("Docker does not seem to be installed")
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
	args = append([]string{"run", "--network=none", "--log-driver=none", "--rm", "-i", img}, args...)
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
