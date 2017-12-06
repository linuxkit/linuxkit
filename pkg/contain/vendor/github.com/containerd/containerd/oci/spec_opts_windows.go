// +build windows

package oci

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/containerd/containerd/containers"
	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/images"
	"github.com/opencontainers/image-spec/specs-go/v1"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

// WithImageConfig configures the spec to from the configuration of an Image
func WithImageConfig(image Image) SpecOpts {
	return func(ctx context.Context, client Client, _ *containers.Container, s *specs.Spec) error {
		ic, err := image.Config(ctx)
		if err != nil {
			return err
		}
		var (
			ociimage v1.Image
			config   v1.ImageConfig
		)
		switch ic.MediaType {
		case v1.MediaTypeImageConfig, images.MediaTypeDockerSchema2Config:
			p, err := content.ReadBlob(ctx, image.ContentStore(), ic.Digest)
			if err != nil {
				return err
			}
			if err := json.Unmarshal(p, &ociimage); err != nil {
				return err
			}
			config = ociimage.Config
		default:
			return fmt.Errorf("unknown image config media type %s", ic.MediaType)
		}
		s.Process.Env = config.Env
		s.Process.Args = append(config.Entrypoint, config.Cmd...)
		s.Process.User = specs.User{
			Username: config.User,
		}
		return nil
	}
}

// WithTTY sets the information on the spec as well as the environment variables for
// using a TTY
func WithTTY(width, height int) SpecOpts {
	return func(_ context.Context, _ Client, _ *containers.Container, s *specs.Spec) error {
		s.Process.Terminal = true
		if s.Process.ConsoleSize == nil {
			s.Process.ConsoleSize = &specs.Box{}
		}
		s.Process.ConsoleSize.Width = uint(width)
		s.Process.ConsoleSize.Height = uint(height)
		return nil
	}
}
