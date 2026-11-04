package docker

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/containerd/containerd/v2/pkg/reference"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	imagespec "github.com/opencontainers/image-spec/specs-go/v1"
)

type readCloser struct {
	r      io.ReadCloser
	closer func() error
}

func (t readCloser) Read(p []byte) (int, error) {
	return t.r.Read(p)
}
func (t readCloser) Close() error {
	return t.closer()
}

// ImageSource a source for an image in the docker engine.
// Implements a moby.ImageSource.
type ImageSource struct {
	ref     *reference.Spec
	context string
}

// NewSource return an ImageSource for a specific ref from docker.
func NewSource(ref *reference.Spec, dockerContext string) ImageSource {
	return ImageSource{
		ref:     ref,
		context: dockerContext,
	}
}

// Config return the imagespec.ImageConfig for the given source.
func (d ImageSource) Config() (imagespec.ImageConfig, error) {
	cli, err := Client(d.context)
	if err != nil {
		return imagespec.ImageConfig{}, err
	}
	defer func() { _ = cli.Close() }()
	inspect, err := InspectImage(cli, d.ref)
	if err != nil {
		return imagespec.ImageConfig{}, fmt.Errorf("docker context %q: inspect image %s: %w", d.context, d.ref, err)
	}
	// because the other parts expect OCI go-spec structs, not dockertypes structs,
	// the easiest way to do this is to convert via json
	configJSON, err := json.Marshal(inspect)
	if err != nil {
		return imagespec.ImageConfig{}, fmt.Errorf("unable to convert image config to json: %v", err)
	}
	var ociConfig imagespec.Image
	err = json.Unmarshal(configJSON, &ociConfig)
	return ociConfig.Config, err
}

// TarReader return an io.ReadCloser to read the filesystem contents of the image.
func (d ImageSource) TarReader() (io.ReadCloser, error) {
	container, err := Create(d.ref.String(), false, d.context)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker image %s: %w", d.ref, err)
	}
	contents, err := Export(container, d.context)
	if err != nil {
		return nil, fmt.Errorf("failed to docker export container from container %s: %w", container, err)
	}

	return readCloser{
		r: contents,
		closer: func() error {
			_ = contents.Close()

			return Rm(container, d.context)
		},
	}, nil
}

// V1TarReader return an io.ReadCloser to read the save of the image
func (d ImageSource) V1TarReader(overrideName string) (io.ReadCloser, error) {
	saveName := d.ref.String()
	if overrideName != "" {
		saveName = overrideName
	}
	return Save(saveName, d.context)
}

// OCITarReader return an io.ReadCloser to read the save of the image
func (d ImageSource) OCITarReader(overrideName string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("unsupported")
}

// Descriptor return the descriptor of the image.
func (d ImageSource) Descriptor() *v1.Descriptor {
	return nil
}

// SBoM not supported in docker, but it is not an error, so just return nil.
func (d ImageSource) SBoMs() ([]io.ReadCloser, error) {
	return nil, nil
}
