package pkglib

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strings"
	"testing"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/reference"
	dockertypes "github.com/docker/docker/api/types"
	registry "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
	lktspec "github.com/linuxkit/linuxkit/src/cmd/linuxkit/spec"
	buildkitClient "github.com/moby/buildkit/client"
	imagespec "github.com/opencontainers/image-spec/specs-go/v1"
)

type dockerMocker struct {
	supportContexts bool
	images          map[string][]byte
	enableTag       bool
	enableBuild     bool
	enablePull      bool
	fixedReadName   string
	builds          []buildLog
}

type buildLog struct {
	tag           string
	pkg           string
	dockerContext string
	platform      string
}

func (d *dockerMocker) tag(ref, tag string) error {
	if !d.enableTag {
		return errors.New("tags not allowed")
	}
	d.images[tag] = d.images[ref]
	return nil
}
func (d *dockerMocker) contextSupportCheck() error {
	if d.supportContexts {
		return nil
	}
	return errors.New("contexts not supported")
}
func (d *dockerMocker) builder(_ context.Context, _, _, _ string, _ bool) (*buildkitClient.Client, error) {
	return nil, fmt.Errorf("not implemented")
}
func (d *dockerMocker) build(ctx context.Context, tag, pkg, dockerContext, builderImage, platform string, builderRestart bool, c lktspec.CacheProvider, r io.Reader, stdout io.Writer, imageBuildOpts dockertypes.ImageBuildOptions) error {
	if !d.enableBuild {
		return errors.New("build disabled")
	}
	d.builds = append(d.builds, buildLog{tag, pkg, dockerContext, platform})
	return nil
}
func (d *dockerMocker) save(tgt string, refs ...string) error {
	var b []byte
	for _, ref := range refs {
		if data, ok := d.images[ref]; ok {
			b = append(b, data...)
			continue
		}
		return fmt.Errorf("do not have image %s", ref)
	}
	return os.WriteFile(tgt, b, 0666)
}
func (d *dockerMocker) load(src io.Reader) error {
	b, err := io.ReadAll(src)
	if err != nil {
		return err
	}
	d.images[d.fixedReadName] = b
	return nil
}
func (d *dockerMocker) pull(img string) (bool, error) {
	if d.enablePull {
		b := make([]byte, 256)
		rand.Read(b)
		d.images[img] = b
		return true, nil
	}
	return false, errors.New("failed to pull")
}

type cacheMocker struct {
	enablePush             bool
	enabledDescriptorWrite bool
	enableImagePull        bool
	enableImageLoad        bool
	enableIndexWrite       bool
	images                 map[string][]registry.Descriptor
	hashes                 map[string][]byte
}

func (c *cacheMocker) ImagePull(ref *reference.Spec, trustedRef, architecture string, alwaysPull bool) (lktspec.ImageSource, error) {
	if !c.enableImagePull {
		return nil, errors.New("ImagePull disabled")
	}
	// make some random data for a layer
	b := make([]byte, 256)
	rand.Read(b)
	return c.imageWriteStream(ref, architecture, bytes.NewReader(b))
}

func (c *cacheMocker) ImageInCache(ref *reference.Spec, trustedRef, architecture string) (bool, error) {
	image := ref.String()
	desc, ok := c.images[image]
	if !ok {
		return false, nil
	}
	for _, d := range desc {
		if d.Platform != nil && d.Platform.Architecture == architecture {
			return true, nil
		}
	}
	return false, nil
}

func (c *cacheMocker) ImageInRegistry(ref *reference.Spec, trustedRef, architecture string) (bool, error) {
	return false, nil
}

func (c *cacheMocker) ImageLoad(ref *reference.Spec, architecture string, r io.Reader) (lktspec.ImageSource, error) {
	if !c.enableImageLoad {
		return nil, errors.New("ImageLoad disabled")
	}
	return c.imageWriteStream(ref, architecture, r)
}

func (c *cacheMocker) imageWriteStream(ref *reference.Spec, architecture string, r io.Reader) (lktspec.ImageSource, error) {
	image := fmt.Sprintf("%s-%s", ref.String(), architecture)

	// make some random data for a layer
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("error reading data: %v", err)
	}
	hash, size, err := registry.SHA256(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("error calculating hash of layer: %v", err)
	}
	c.assignHash(hash.String(), b)

	im := registry.Manifest{
		MediaType: types.OCIManifestSchema1,
		Layers: []registry.Descriptor{
			{MediaType: types.OCILayer, Size: size, Digest: hash},
		},
		SchemaVersion: 2,
	}

	// write the updated index, remove the old one
	b, err = json.Marshal(im)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal new image to json: %v", err)
	}
	hash, size, err = registry.SHA256(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("error calculating hash of index json: %v", err)
	}
	c.assignHash(hash.String(), b)
	desc := registry.Descriptor{
		MediaType: types.OCIManifestSchema1,
		Size:      size,
		Digest:    hash,
		Annotations: map[string]string{
			imagespec.AnnotationRefName: image,
		},
		Platform: &registry.Platform{
			OS:           "linux",
			Architecture: architecture,
		},
	}
	c.appendImage(image, desc)

	return c.NewSource(ref, architecture, &desc), nil
}

func (c *cacheMocker) IndexWrite(ref *reference.Spec, descriptors ...registry.Descriptor) (lktspec.ImageSource, error) {
	if !c.enableIndexWrite {
		return nil, errors.New("disabled")
	}
	image := ref.String()
	im := registry.IndexManifest{
		MediaType:     types.OCIImageIndex,
		Manifests:     descriptors,
		SchemaVersion: 2,
	}

	// write the updated index, remove the old one
	b, err := json.Marshal(im)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal new index to json: %v", err)
	}
	hash, size, err := registry.SHA256(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("error calculating hash of index json: %v", err)
	}
	c.assignHash(hash.String(), b)
	desc := registry.Descriptor{
		MediaType: types.OCIImageIndex,
		Size:      size,
		Digest:    hash,
		Annotations: map[string]string{
			imagespec.AnnotationRefName: image,
		},
	}
	c.appendImage(image, desc)

	return c.NewSource(ref, "", &desc), nil
}
func (c *cacheMocker) Push(name string, withManifest bool) error {
	if !c.enablePush {
		return errors.New("push disabled")
	}
	if _, ok := c.images[name]; !ok {
		return fmt.Errorf("unknown image %s", name)
	}
	return nil
}

func (c *cacheMocker) DescriptorWrite(ref *reference.Spec, desc registry.Descriptor) (lktspec.ImageSource, error) {
	if !c.enabledDescriptorWrite {
		return nil, errors.New("descriptor disabled")
	}
	var (
		image = ref.String()
		im    = registry.IndexManifest{
			MediaType:     types.OCIImageIndex,
			Manifests:     []registry.Descriptor{desc},
			SchemaVersion: 2,
		}
	)
	// write the updated index, remove the old one
	b, err := json.Marshal(im)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal new index to json: %v", err)
	}
	hash, size, err := registry.SHA256(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("error calculating hash of index json: %v", err)
	}
	c.assignHash(hash.String(), b)
	root := registry.Descriptor{
		MediaType: types.OCIImageIndex,
		Size:      size,
		Digest:    hash,
		Annotations: map[string]string{
			imagespec.AnnotationRefName: image,
		},
	}
	c.appendImage(image, root)

	return c.NewSource(ref, "", &root), nil
}
func (c *cacheMocker) FindDescriptor(ref *reference.Spec) (*registry.Descriptor, error) {
	name := ref.String()
	if desc, ok := c.images[name]; ok && len(desc) > 0 {
		return &desc[0], nil
	}
	return nil, fmt.Errorf("not found %s", name)
}
func (c *cacheMocker) NewSource(ref *reference.Spec, architecture string, descriptor *registry.Descriptor) lktspec.ImageSource {
	return cacheMockerSource{c, ref, architecture, descriptor}
}
func (c *cacheMocker) assignHash(hash string, b []byte) {
	if c.hashes == nil {
		c.hashes = map[string][]byte{}
	}
	c.hashes[hash] = b
}
func (c *cacheMocker) appendImage(image string, root registry.Descriptor) {
	if c.images == nil {
		c.images = map[string][]registry.Descriptor{}
	}
	c.images[image] = append(c.images[image], root)
}

// Store get content.Store referencing the cache
func (c *cacheMocker) Store() (content.Store, error) {
	return nil, errors.New("unsupported")
}

type cacheMockerSource struct {
	c            *cacheMocker
	ref          *reference.Spec
	architecture string
	descriptor   *registry.Descriptor
}

func (c cacheMockerSource) Config() (imagespec.ImageConfig, error) {
	return imagespec.ImageConfig{}, errors.New("unsupported")
}
func (c cacheMockerSource) TarReader() (io.ReadCloser, error) {
	return nil, errors.New("unsupported")
}
func (c cacheMockerSource) V1TarReader(overrideName string) (io.ReadCloser, error) {
	_, found := c.c.images[c.ref.String()]
	if !found {
		return nil, fmt.Errorf("no image found with ref: %s", c.ref.String())
	}
	b := make([]byte, 256)
	rand.Read(b)
	return io.NopCloser(bytes.NewReader(b)), nil
}
func (c cacheMockerSource) Descriptor() *registry.Descriptor {
	return c.descriptor
}

func TestBuild(t *testing.T) {
	var (
		cacheDir = "somecachedir"
	)
	tests := []struct {
		msg     string
		p       Pkg
		options []BuildOpt
		targets []string
		runner  *dockerMocker
		cache   *cacheMocker
		err     string
	}{
		{"invalid tag", Pkg{image: "docker.io/foo/bar:abc:def:ghi"}, nil, nil, &dockerMocker{}, &cacheMocker{}, "could not resolve references"},
		{"not at head", Pkg{org: "foo", image: "bar", hash: "abc", arches: []string{"amd64"}, commitHash: "foo"}, nil, []string{"amd64"}, &dockerMocker{supportContexts: false}, &cacheMocker{}, "Cannot build from commit hash != HEAD"},
		{"no build cache", Pkg{org: "foo", image: "bar", hash: "abc", arches: []string{"amd64"}, commitHash: "HEAD"}, nil, []string{"amd64"}, &dockerMocker{supportContexts: false}, &cacheMocker{}, "must provide linuxkit build cache"},
		{"unsupported contexts", Pkg{org: "foo", image: "bar", hash: "abc", arches: []string{"amd64"}, commitHash: "HEAD"}, []BuildOpt{WithBuildCacheDir(cacheDir)}, []string{"amd64"}, &dockerMocker{supportContexts: false}, &cacheMocker{}, "contexts not supported, check docker version"},
		{"load docker without local platform", Pkg{org: "foo", image: "bar", hash: "abc", arches: []string{"amd64", "arm64"}, commitHash: "HEAD"}, []BuildOpt{WithBuildCacheDir(cacheDir), WithBuildTargetDockerCache()}, []string{"amd64", "arm64"}, &dockerMocker{supportContexts: true, enableBuild: true, images: map[string][]byte{}, enableTag: true}, &cacheMocker{enableImagePull: false, enableImageLoad: true, enableIndexWrite: true}, ""},
		{"amd64", Pkg{org: "foo", image: "bar", hash: "abc", arches: []string{"amd64", "arm64"}, commitHash: "HEAD"}, []BuildOpt{WithBuildCacheDir(cacheDir)}, []string{"amd64"}, &dockerMocker{supportContexts: true, enableBuild: true}, &cacheMocker{enableImagePull: false, enableImageLoad: true, enableIndexWrite: true}, ""},
		{"arm64", Pkg{org: "foo", image: "bar", hash: "abc", arches: []string{"amd64", "arm64"}, commitHash: "HEAD"}, []BuildOpt{WithBuildCacheDir(cacheDir)}, []string{"arm64"}, &dockerMocker{supportContexts: true, enableBuild: true}, &cacheMocker{enableImagePull: false, enableImageLoad: true, enableIndexWrite: true}, ""},
		{"amd64 and arm64", Pkg{org: "foo", image: "bar", hash: "abc", arches: []string{"amd64", "arm64"}, commitHash: "HEAD"}, []BuildOpt{WithBuildCacheDir(cacheDir)}, []string{"amd64", "arm64"}, &dockerMocker{supportContexts: true, enableBuild: true}, &cacheMocker{enableImagePull: false, enableImageLoad: true, enableIndexWrite: true}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			opts := append(tt.options, WithBuildDocker(tt.runner), WithBuildCacheProvider(tt.cache), WithBuildOutputWriter(io.Discard))
			// build our build options
			if len(tt.targets) > 0 {
				var targets []imagespec.Platform
				for _, arch := range tt.targets {
					targets = append(targets, imagespec.Platform{OS: "linux", Architecture: arch})
				}
				opts = append(opts, WithBuildPlatforms(targets...))
			}
			err := tt.p.Build(opts...)
			switch {
			case (tt.err == "" && err != nil) || (tt.err != "" && err == nil) || (tt.err != "" && err != nil && !strings.HasPrefix(err.Error(), tt.err)):
				t.Errorf("mismatched errors actual '%v', expected '%v'", err, tt.err)
			case tt.err == "" && len(tt.runner.builds) != len(tt.targets):
				// need to make sure that it was called the correct number of times with the correct arguments
				t.Errorf("mismatched call to runners, should be %d was %d: %#v", len(tt.targets), len(tt.runner.builds), tt.runner.builds)
			case tt.err == "":
				// check that all of our platforms were called exactly once each
				// we do that by:
				// 1- creating a map of all of the target platforms and setting them to `false`
				// 2- checking with each build for which platform it was called
				//
				// each build is assumed to track what platform it built
				platformMap := map[string]bool{}
				for _, arch := range tt.targets {
					platformMap[fmt.Sprintf("linux/%s", arch)] = false
				}
				for _, build := range tt.runner.builds {
					if err := testCheckBuildRun(build, platformMap); err != nil {
						t.Errorf("mismatch in build: '%v', %#v", err, build)
					}
				}
				for k, v := range platformMap {
					if !v {
						t.Errorf("did not execute build for platform: %s", k)
					}
				}
			}
		})
	}
}

// testCheckBuildRun check the output of a build run
func testCheckBuildRun(build buildLog, platforms map[string]bool) error {
	platform := build.platform
	used, ok := platforms[platform]
	if !ok {
		return fmt.Errorf("requested unknown platform: %s", platform)
	}
	if used {
		return fmt.Errorf("tried to use platform twice: %s", platform)
	}
	platforms[platform] = true
	return nil
}
