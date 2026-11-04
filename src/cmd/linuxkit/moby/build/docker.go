package build

import (
	"context"
	"io"

	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/dockercontext"
)

// dockerRun pins both pull and run to the same explicitly resolved context.
// runEnv supplies container environment variables, not endpoint overrides.
func dockerRun(dockerContext string, input io.Reader, output io.Writer, img string, runEnv []string, imageArgs ...string) error {
	selection, err := dockercontext.Resolve(dockerContext)
	if err != nil {
		return err
	}
	if err := selection.Run(context.Background(), nil, nil, nil, "pull", img); err != nil {
		return err
	}
	args := []string{"run", "--network=none", "--log-driver=none", "--rm", "-i"}
	for _, e := range runEnv {
		args = append(args, "-e", e)
	}
	args = append(args, img)
	args = append(args, imageArgs...)
	return selection.Run(context.Background(), input, output, nil, args...)
}
