package cache

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/containerd/containerd/reference"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/match"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/google/go-containerregistry/pkg/v1/types"
	intoto "github.com/in-toto/in-toto-golang/in_toto"
	lktspec "github.com/linuxkit/linuxkit/src/cmd/linuxkit/spec"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/util"
	imagespec "github.com/opencontainers/image-spec/specs-go/v1"
)

const (
	inTotoJsonMediaType = "application/vnd.in-toto+json"
	layoutFile          = `{
		"imageLayoutVersion": "1.0.0"
	}`
)

// ImageSource a source for an image in the OCI distribution cache.
// Implements a spec.ImageSource.
type ImageSource struct {
	ref        *reference.Spec
	provider   *Provider
	platform   *imagespec.Platform
	descriptor *v1.Descriptor
}

type spdxStatement struct {
	intoto.StatementHeader
	Predicate json.RawMessage `json:"predicate"`
}

// NewSource return an ImageSource for a specific ref and architecture in the given
// cache directory.
func (p *Provider) NewSource(ref *reference.Spec, platform *imagespec.Platform, descriptor *v1.Descriptor) lktspec.ImageSource {
	return ImageSource{
		ref:        ref,
		provider:   p,
		platform:   platform,
		descriptor: descriptor,
	}
}

// Config return the imagespec.ImageConfig for the given source. Resolves to the
// architecture, if necessary.
func (c ImageSource) Config() (imagespec.ImageConfig, error) {
	imageName := c.ref.String()
	image, err := c.provider.findImage(imageName, *c.platform)
	if err != nil {
		return imagespec.ImageConfig{}, err
	}

	configFile, err := image.ConfigFile()
	if err != nil {
		return imagespec.ImageConfig{}, fmt.Errorf("unable to get image OCI ConfigFile: %v", err)
	}
	// because the other parts expect OCI go-spec structs, not google/go-containerregistry structs,
	// the easiest way to do this is to convert via json
	configJSON, err := json.Marshal(configFile.Config)
	if err != nil {
		return imagespec.ImageConfig{}, fmt.Errorf("unable to convert image config to json: %v", err)
	}
	var ociConfig imagespec.ImageConfig
	err = json.Unmarshal(configJSON, &ociConfig)
	return ociConfig, err
}

// TarReader return an io.ReadCloser to read the filesystem contents of the image,
// as resolved to the provided architecture.
func (c ImageSource) TarReader() (io.ReadCloser, error) {
	imageName := c.ref.String()

	// get a reference to the image
	image, err := c.provider.findImage(imageName, *c.platform)
	if err != nil {
		return nil, err
	}

	return mutate.Extract(image), nil
}

// V1TarReader return an io.ReadCloser to read the image as a v1 tarball
func (c ImageSource) V1TarReader(overrideName string) (io.ReadCloser, error) {
	imageName := c.ref.String()
	saveName := imageName
	if overrideName != "" {
		saveName = overrideName
	}
	refName, err := name.ParseReference(saveName)
	if err != nil {
		return nil, fmt.Errorf("error parsing image name: %v", err)
	}
	// get a reference to the image
	image, err := c.provider.findImage(imageName, *c.platform)
	if err != nil {
		return nil, err
	}
	// convert the writer to a reader
	r, w := io.Pipe()
	go func() {
		defer w.Close()
		_ = tarball.Write(refName, image, w)
	}()
	return r, nil
}

// OCITarReader return an io.ReadCloser to read the image as a v1 tarball
func (c ImageSource) OCITarReader(overrideName string) (io.ReadCloser, error) {
	imageName := c.ref.String()
	saveName := imageName
	if overrideName != "" {
		saveName = overrideName
	}
	refName, err := name.ParseReference(saveName)
	if err != nil {
		return nil, fmt.Errorf("error parsing image name: %v", err)
	}
	// get a reference to the image
	image, err := c.provider.findImage(imageName, *c.platform)
	if err != nil {
		return nil, err
	}
	// convert the writer to a reader
	r, w := io.Pipe()
	go func() {
		defer w.Close()
		tw := tar.NewWriter(w)
		defer tw.Close()
		if err := writeLayoutHeader(tw); err != nil {
			_ = w.CloseWithError(err)
			return
		}

		if err := writeLayoutImage(tw, image); err != nil {
			_ = w.CloseWithError(err)
			return
		}

		imageDigest, err := image.Digest()
		if err != nil {
			_ = w.CloseWithError(err)
			return
		}
		imageSize, err := image.Size()
		if err != nil {
			_ = w.CloseWithError(err)
			return
		}

		// write the index file
		desc := v1.Descriptor{
			MediaType: types.OCIImageIndex,
			Size:      imageSize,
			Digest:    imageDigest,
			Annotations: map[string]string{
				imagespec.AnnotationRefName: refName.String(),
			},
		}
		if err := writeLayoutIndex(tw, desc); err != nil {
			_ = w.CloseWithError(err)
			return
		}
	}()
	return r, nil
}

// Descriptor return the descriptor of the image.
func (c ImageSource) Descriptor() *v1.Descriptor {
	return c.descriptor
}

// SBoM return the sbom for the image
func (c ImageSource) SBoMs() ([]io.ReadCloser, error) {
	index, err := c.provider.findIndex(c.ref.String())
	// if it is not an index, we actually do not care much
	if err != nil {
		return nil, nil
	}

	// get the digest of the manifest that represents our targeted architecture
	descs, err := partial.FindManifests(index, matchPlatformsOSArch(v1.Platform{OS: c.platform.OS, Architecture: c.platform.Architecture}))
	if err != nil {
		return nil, err
	}
	if len(descs) < 1 {
		return nil, fmt.Errorf("no manifest found for %s platform %s", c.ref.String(), c.platform)
	}
	if len(descs) > 1 {
		return nil, fmt.Errorf("multiple manifests found for %s platform %s", c.ref.String(), c.platform)
	}
	// get the digest of the manifest that represents our targeted architecture
	desc := descs[0]

	annotations := map[string]string{
		util.AnnotationDockerReferenceType:   util.AnnotationAttestationManifest,
		util.AnnotationDockerReferenceDigest: desc.Digest.String(),
	}
	descs, err = partial.FindManifests(index, matchAllAnnotations(annotations))
	if err != nil {
		return nil, err
	}
	if len(descs) > 1 {
		return nil, fmt.Errorf("multiple manifests found for %s platform %s", c.ref.String(), c.platform)
	}
	if len(descs) < 1 {
		return nil, nil
	}

	// get the layers for the first descriptor
	images, err := partial.FindImages(index, match.Digests(descs[0].Digest))
	if err != nil {
		return nil, err
	}
	if len(images) < 1 {
		return nil, fmt.Errorf("no attestation image found for %s platform %s, even though the manifest exists", c.ref.String(), c.platform)
	}
	if len(images) > 1 {
		return nil, fmt.Errorf("multiple attestation images found for %s platform %s", c.ref.String(), c.platform)
	}
	image := images[0]
	manifest, err := image.Manifest()
	if err != nil {
		return nil, err
	}
	layers, err := image.Layers()
	if err != nil {
		return nil, err
	}
	if len(manifest.Layers) != len(layers) {
		return nil, fmt.Errorf("manifest layers and image layers do not match for the attestation for %s platform %s", c.ref.String(), c.platform)
	}
	var readers []io.ReadCloser
	for i, layer := range manifest.Layers {
		annotations := layer.Annotations
		if annotations[util.AnnotationInTotoPredicateType] != util.AnnotationSPDXDoc || layer.MediaType != inTotoJsonMediaType {
			continue
		}
		// get the actual blob of the layer
		layer, err := layers[i].Compressed()
		if err != nil {
			return nil, err
		}
		// read the layer, we want just the predicate, stripping off the header
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, layer); err != nil {
			return nil, err
		}
		layer.Close()
		var stmt spdxStatement
		if err := json.Unmarshal(buf.Bytes(), &stmt); err != nil {
			return nil, err
		}
		if stmt.PredicateType != util.AnnotationSPDXDoc {
			return nil, fmt.Errorf("unexpected predicate type %s", stmt.PredicateType)
		}
		sbom := stmt.Predicate

		readers = append(readers, io.NopCloser(bytes.NewReader(sbom)))
	}
	// get the content of the single descriptor
	return readers, nil
}
