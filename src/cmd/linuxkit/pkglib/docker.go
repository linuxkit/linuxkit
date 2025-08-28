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

type dockerRunner interface {
	tag(ref, tag string) error
	build(ctx context.Context, tag, pkg, dockerContext, builderImage, builderConfigPath, platform string, restart, preCacheImages bool, c spec.CacheProvider, r io.Reader, stdout io.Writer, sbomScan bool, sbomScannerImage, platformType string, imageBuildOpts spec.ImageBuildOptions) error
	save(tgt string, refs ...string) error
	load(src io.Reader) error
	pull(img string) (bool, error)
	contextSupportCheck() error
	builder(ctx context.Context, dockerContext, builderImage, builderConfigPath, platform string, restart bool) (*buildkitClient.Client, error)
}
