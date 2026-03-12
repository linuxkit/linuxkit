package pkglib

// Thin wrappers around Docker CLI invocations

//go:generate ./gen

import (
	"context"
	"io"

	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/spec"
	buildkitClient "github.com/moby/buildkit/client"

	// golint requires comments on non-main(test)
	// package for blank import

	_ "github.com/moby/buildkit/client/connhelper/dockercontainer"
	_ "github.com/moby/buildkit/client/connhelper/ssh"
)

// BuilderConfig holds configuration for the buildkit builder container.
type BuilderConfig struct {
	// Name is the container name for the buildkit builder (e.g., "linuxkit-builder-alice").
	Name string
	// Image is the container image to run (e.g., "moby/buildkit:v0.26.3").
	Image string
	// ConfigPath is an optional path to a buildkitd.toml config file.
	ConfigPath string
	// Restart forces recreation of the builder container even if one with the
	// correct name and image already exists.
	Restart bool
}

type DockerRunner interface {
	Tag(ref, tag string) error
	Build(ctx context.Context, tag, pkg, dockerContext, platform string, preCacheImages bool, c spec.CacheProvider, r io.Reader, stdout io.Writer, sbomScan bool, sbomScannerImage, platformType string, imageBuildOpts spec.ImageBuildOptions) error
	Save(tgt string, refs ...string) error
	Load(src io.Reader) error
	Pull(img string) (bool, error)
	ContextSupportCheck() error
	Builder(ctx context.Context, dockerContext, platform string) (*buildkitClient.Client, error)
}
