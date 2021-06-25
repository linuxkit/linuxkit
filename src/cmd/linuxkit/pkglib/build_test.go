package pkglib

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"runtime"
	"strings"
	"testing"

	"github.com/containerd/containerd/reference"
	registry "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
	lktspec "github.com/linuxkit/linuxkit/src/cmd/linuxkit/spec"
	imagespec "github.com/opencontainers/image-spec/specs-go/v1"
)

type dockerMocker struct {
	supportBuildKit bool
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
	opts          []string
}

func (d *dockerMocker) buildkitCheck() error {
	if d.supportBuildKit {
		return nil
	}
	return errors.New("buildkit unsupported")
}
func (d *dockerMocker) tag(ref, tag string) error {
	if !d.enableTag {
		return errors.New("tags not allowed")
	}
	d.images[tag] = d.images[ref]
	return nil
}
func (d *dockerMocker) build(tag, pkg, dockerContext, platform string, stdin io.Reader, stdout io.Writer, opts ...string) error {
	if !d.enableBuild {
		return errors.New("build disabled")
	}
	d.builds = append(d.builds, buildLog{tag, pkg, dockerContext, platform, opts})
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
	return ioutil.WriteFile(tgt, b, 0666)
}
func (d *dockerMocker) load(src io.Reader) error {
	b, err := ioutil.ReadAll(src)
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

func (c *cacheMocker) ImageLoad(ref *reference.Spec, architecture string, r io.Reader) (lktspec.ImageSource, error) {
	if !c.enableImageLoad {
		return nil, errors.New("ImageLoad disabled")
	}
	return c.imageWriteStream(ref, architecture, r)
}

func (c *cacheMocker) imageWriteStream(ref *reference.Spec, architecture string, r io.Reader) (lktspec.ImageSource, error) {
	image := ref.String()

	// make some random data for a layer
	b, err := ioutil.ReadAll(r)
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
	}
	c.appendImage(image, desc)

	return c.NewSource(ref, "", &desc), nil
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
func (c *cacheMocker) Push(name string) error {
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
func (c *cacheMocker) FindDescriptor(name string) (*registry.Descriptor, error) {
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
func (c cacheMockerSource) V1TarReader() (io.ReadCloser, error) {
	return nil, errors.New("unsupported")
}
func (c cacheMockerSource) Descriptor() *registry.Descriptor {
	return c.descriptor
}

func TestBuild(t *testing.T) {
	var (
		nonLocal string
		cacheDir = "somecachedir"
	)
	if runtime.GOARCH == "amd64" {
		nonLocal = "arm64"
	} else {
		nonLocal = "amd64"
	}
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
		{"not at head", Pkg{org: "foo", image: "bar", hash: "abc", arches: []string{"amd64"}, commitHash: "foo"}, nil, []string{"amd64"}, &dockerMocker{supportBuildKit: false}, &cacheMocker{}, "Cannot build from commit hash != HEAD"},
		{"no build cache", Pkg{org: "foo", image: "bar", hash: "abc", arches: []string{"amd64"}, commitHash: "HEAD"}, nil, []string{"amd64"}, &dockerMocker{supportBuildKit: false}, &cacheMocker{}, "must provide linuxkit build cache"},
		{"unsupported buildkit", Pkg{org: "foo", image: "bar", hash: "abc", arches: []string{"amd64"}, commitHash: "HEAD"}, []BuildOpt{WithBuildCacheDir(cacheDir)}, []string{"amd64"}, &dockerMocker{supportBuildKit: false}, &cacheMocker{}, "buildkit not supported, check docker version"},
		{"load docker without local platform", Pkg{org: "foo", image: "bar", hash: "abc", arches: []string{"amd64", "arm64"}, commitHash: "HEAD"}, []BuildOpt{WithBuildCacheDir(cacheDir), WithBuildTargetDockerCache()}, []string{nonLocal}, &dockerMocker{supportBuildKit: false}, &cacheMocker{}, "must build for local platform"},
		{"amd64", Pkg{org: "foo", image: "bar", hash: "abc", arches: []string{"amd64", "arm64"}, commitHash: "HEAD"}, []BuildOpt{WithBuildCacheDir(cacheDir)}, []string{"amd64"}, &dockerMocker{supportBuildKit: true, enableBuild: true}, &cacheMocker{enableImagePull: false, enableImageLoad: true, enableIndexWrite: true}, ""},
		{"arm64", Pkg{org: "foo", image: "bar", hash: "abc", arches: []string{"amd64", "arm64"}, commitHash: "HEAD"}, []BuildOpt{WithBuildCacheDir(cacheDir)}, []string{"arm64"}, &dockerMocker{supportBuildKit: true, enableBuild: true}, &cacheMocker{enableImagePull: false, enableImageLoad: true, enableIndexWrite: true}, ""},
		{"amd64 and arm64", Pkg{org: "foo", image: "bar", hash: "abc", arches: []string{"amd64", "arm64"}, commitHash: "HEAD"}, []BuildOpt{WithBuildCacheDir(cacheDir)}, []string{"amd64", "arm64"}, &dockerMocker{supportBuildKit: true, enableBuild: true}, &cacheMocker{enableImagePull: false, enableImageLoad: true, enableIndexWrite: true}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			opts := append(tt.options, WithBuildDocker(tt.runner), WithBuildCacheProvider(tt.cache), WithBuildOutputWriter(ioutil.Discard))
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
				// check that all of our platforms were called
				platformMap := map[string]bool{}
				for _, arch := range tt.targets {
					platformMap[fmt.Sprintf("linux/%s", arch)] = false
				}
				for _, build := range tt.runner.builds {
					if err := testCheckBuildRun(build, platformMap); err != nil {
						t.Errorf("mismatch in build: '%v', %#v", err, build)
					}
				}
			}
		})
	}
}

// testCheckBuildRun check the output of a build run
func testCheckBuildRun(build buildLog, platforms map[string]bool) error {
	for i, arg := range build.opts {
		switch {
		case arg == "--platform", arg == "-platform":
			if i+1 >= len(build.opts) {
				return errors.New("provided arg --platform with no next argument")
			}
			platform := build.opts[i+1]
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
	}
	return errors.New("missing platform argument")
}
