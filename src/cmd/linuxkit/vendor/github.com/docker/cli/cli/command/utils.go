// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.23

package command

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/streams"
	"github.com/docker/cli/internal/prompt"
	"github.com/docker/docker/api/types/filters"
	"github.com/moby/sys/atomicwriter"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
)

// CopyToFile writes the content of the reader to the specified file
//
// Deprecated: use [atomicwriter.New].
func CopyToFile(outfile string, r io.Reader) error {
	writer, err := atomicwriter.New(outfile, 0o600)
	if err != nil {
		return err
	}
	defer writer.Close()
	_, err = io.Copy(writer, r)
	return err
}

const ErrPromptTerminated = prompt.ErrTerminated

// DisableInputEcho disables input echo on the provided streams.In.
// This is useful when the user provides sensitive information like passwords.
// The function returns a restore function that should be called to restore the
// terminal state.
func DisableInputEcho(ins *streams.In) (restore func() error, err error) {
	return prompt.DisableInputEcho(ins)
}

// PromptForInput requests input from the user.
//
// If the user terminates the CLI with SIGINT or SIGTERM while the prompt is
// active, the prompt will return an empty string ("") with an ErrPromptTerminated error.
// When the prompt returns an error, the caller should propagate the error up
// the stack and close the io.Reader used for the prompt which will prevent the
// background goroutine from blocking indefinitely.
func PromptForInput(ctx context.Context, in io.Reader, out io.Writer, message string) (string, error) {
	return prompt.ReadInput(ctx, in, out, message)
}

// PromptForConfirmation requests and checks confirmation from the user.
// This will display the provided message followed by ' [y/N] '. If the user
// input 'y' or 'Y' it returns true otherwise false. If no message is provided,
// "Are you sure you want to proceed? [y/N] " will be used instead.
//
// If the user terminates the CLI with SIGINT or SIGTERM while the prompt is
// active, the prompt will return false with an ErrPromptTerminated error.
// When the prompt returns an error, the caller should propagate the error up
// the stack and close the io.Reader used for the prompt which will prevent the
// background goroutine from blocking indefinitely.
func PromptForConfirmation(ctx context.Context, ins io.Reader, outs io.Writer, message string) (bool, error) {
	return prompt.Confirm(ctx, ins, outs, message)
}

// PruneFilters merges prune filters specified in config.json with those specified
// as command-line flags.
//
// CLI label filters have precedence over those specified in config.json. If a
// label filter specified as flag conflicts with a label defined in config.json
// (i.e., "label=some-value" conflicts with "label!=some-value", and vice versa),
// then the filter defined in config.json is omitted.
func PruneFilters(dockerCLI config.Provider, pruneFilters filters.Args) filters.Args {
	cfg := dockerCLI.ConfigFile()
	if cfg == nil {
		return pruneFilters
	}

	// Merge filters provided through the CLI with default filters defined
	// in the CLI-configfile.
	for _, f := range cfg.PruneFilters {
		k, v, ok := strings.Cut(f, "=")
		if !ok {
			continue
		}
		switch k {
		case "label":
			// "label != some-value" conflicts with "label = some-value"
			if pruneFilters.ExactMatch("label!", v) {
				continue
			}
			pruneFilters.Add(k, v)
		case "label!":
			// "label != some-value" conflicts with "label = some-value"
			if pruneFilters.ExactMatch("label", v) {
				continue
			}
			pruneFilters.Add(k, v)
		default:
			pruneFilters.Add(k, v)
		}
	}

	return pruneFilters
}

// AddPlatformFlag adds `platform` to a set of flags for API version 1.32 and later.
func AddPlatformFlag(flags *pflag.FlagSet, target *string) {
	flags.StringVar(target, "platform", os.Getenv("DOCKER_DEFAULT_PLATFORM"), "Set platform if server is multi-platform capable")
	_ = flags.SetAnnotation("platform", "version", []string{"1.32"})
}

// ValidateOutputPath validates the output paths of the "docker cp" command.
func ValidateOutputPath(path string) error {
	dir := filepath.Dir(filepath.Clean(path))
	if dir != "" && dir != "." {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return errors.Errorf("invalid output path: directory %q does not exist", dir)
		}
	}
	// check whether `path` points to a regular file
	// (if the path exists and doesn't point to a directory)
	if fileInfo, err := os.Stat(path); !os.IsNotExist(err) {
		if err != nil {
			return err
		}

		if fileInfo.Mode().IsDir() || fileInfo.Mode().IsRegular() {
			return nil
		}

		if err := ValidateOutputPathFileMode(fileInfo.Mode()); err != nil {
			return errors.Wrapf(err, "invalid output path: %q must be a directory or a regular file", path)
		}
	}
	return nil
}

// ValidateOutputPathFileMode validates the output paths of the "docker cp" command
// and serves as a helper to [ValidateOutputPath]
func ValidateOutputPathFileMode(fileMode os.FileMode) error {
	switch {
	case fileMode&os.ModeDevice != 0:
		return errors.New("got a device")
	case fileMode&os.ModeIrregular != 0:
		return errors.New("got an irregular file")
	}
	return nil
}

func invalidParameter(err error) error {
	return invalidParameterErr{err}
}

type invalidParameterErr struct{ error }

func (invalidParameterErr) InvalidParameter() {}

func notFound(err error) error {
	return notFoundErr{err}
}

type notFoundErr struct{ error }

func (notFoundErr) NotFound() {}
func (e notFoundErr) Unwrap() error {
	return e.error
}
