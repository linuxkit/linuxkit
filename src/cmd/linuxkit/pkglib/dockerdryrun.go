package pkglib

// Thin wrappers around Docker CLI invocations

//go:generate ./gen

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/spec"
	buildkitClient "github.com/moby/buildkit/client"

	// golint requires comments on non-main(test)
	// package for blank import

	_ "github.com/moby/buildkit/client/connhelper/dockercontainer"
	_ "github.com/moby/buildkit/client/connhelper/ssh"
)

type dockerDryRunnerImpl struct {
}

func NewDockerDryRunner() DockerRunner {
	return &dockerDryRunnerImpl{}
}

func (dr *dockerDryRunnerImpl) ContextSupportCheck() error {
	return nil
}

// builder ensure that a builder container exists or return an error.
//
// Process:
//
// 1. Get an appropriate docker context.
// 2. Using the appropriate context, try to find a docker container named `linuxkit-builder` in that context.
// 3. Return a reference to that container.
//
// To get the appropriate docker context:
//
// 1. if dockerContext is provided, try to create a builder with that context; if it succeeds, we are done; if not, return an error.
// 2. try to find an existing named runner with the pattern; if it succeeds, we are done; if not, try next.
// 3. try to create a generic builder using the default context named "linuxkit".
func (dr *dockerDryRunnerImpl) Builder(ctx context.Context, dockerContext, builderImage, builderConfigPath, platform string, restart bool) (*buildkitClient.Client, error) {
	return nil, nil
}

func (dr *dockerDryRunnerImpl) Pull(img string) (bool, error) {
	return false, errors.New("not implemented")
}

func (dr *dockerDryRunnerImpl) Tag(ref, tag string) error {
	return errors.New("not implemented")
}

func (dr *dockerDryRunnerImpl) Build(ctx context.Context, tag, pkg, dockerContext, builderImage, builderConfigPath, platform string, restart, preCacheImages bool, c spec.CacheProvider, stdin io.Reader, stdout io.Writer, sbomScan bool, sbomScannerImage, progressType string, imageBuildOpts spec.ImageBuildOptions) error {
	// build args
	var buildArgs []string
	for k, v := range imageBuildOpts.BuildArgs {
		buildArgs = append(buildArgs, fmt.Sprintf("--build-arg=%s=%s", k, *v))
	}

	// network
	var network string
	switch imageBuildOpts.NetworkMode {
	case "host", "none", "default":
		network = fmt.Sprintf("--network=%s", imageBuildOpts.NetworkMode)
	default:
		return fmt.Errorf("unsupported network mode %q", imageBuildOpts.NetworkMode)
	}

	// labels
	var labels []string
	for k, v := range imageBuildOpts.Labels {
		labels = append(labels, fmt.Sprintf("--label=%s=%s", k, v))
	}

	fmt.Printf("docker buildx build --platform %s -t %s -f %s %s %s %s %s\n", platform, tag, path.Join(pkg, imageBuildOpts.Dockerfile), strings.Join(buildArgs, " "), strings.Join(labels, " "), network, pkg)

	return nil
}

func (dr *dockerDryRunnerImpl) Save(tgt string, refs ...string) error {
	return errors.New("not implemented")
}

func (dr *dockerDryRunnerImpl) Load(src io.Reader) error {
	return errors.New("not implemented")
}
