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

type DockerRunner interface {
	Tag(ref, tag string) error
	Build(ctx context.Context, tag, pkg, dockerContext, builderImage, builderConfigPath, platform string, restart, preCacheImages bool, c spec.CacheProvider, r io.Reader, stdout io.Writer, sbomScan bool, sbomScannerImage, platformType string, imageBuildOpts spec.ImageBuildOptions) error
	Save(tgt string, refs ...string) error
	Load(src io.Reader) error
	Pull(img string) (bool, error)
	ContextSupportCheck() error
	Builder(ctx context.Context, dockerContext, builderImage, builderConfigPath, platform string, restart bool) (*buildkitClient.Client, error)
}
