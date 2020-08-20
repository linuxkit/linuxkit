package testhelper

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// DockerRun run a docker container
// thanks to moby/tool, which is licensed apache 2.0
func DockerRun(input io.Reader, output io.Writer, trust bool, rm bool, img string, args ...string) error {
	docker, err := exec.LookPath("docker")
	if err != nil {
		return errors.New("Docker does not seem to be installed")
	}

	env := os.Environ()
	if trust {
		env = append(env, "DOCKER_CONTENT_TRUST=1")
	}

	dArgs := []string{"run", "--network=none"}
	if rm {
		dArgs = append(dArgs, "--rm")
	}
	dArgs = append(dArgs, "-i", img)
	dArgs = append(dArgs, args...)
	cmd := exec.Command(docker, dArgs...)
	cmd.Stdin = input
	cmd.Stdout = output
	cmd.Stderr = output
	cmd.Env = env
	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("docker run failed: %v output:\n%s", err, exitError.Stderr)
		}
		return err
	}

	return nil
}
