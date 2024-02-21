package pkglib

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/reference"
	dockertypes "github.com/docker/docker/api/types"
	registry "github.com/google/go-containerregistry/pkg/v1"
	v1 "github.com/google/go-containerregistry/pkg/v1"
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
func (d *dockerMocker) build(ctx context.Context, tag, pkg, dockerfile, dockerContext, builderImage, platform string, builderRestart bool, c lktspec.CacheProvider, r io.Reader, stdout io.Writer, sbomScan bool, sbomScannerImage string, imageBuildOpts dockertypes.ImageBuildOptions) error {
	if !d.enableBuild {
		return errors.New("build disabled")
	}
	d.builds = append(d.builds, buildLog{tag, pkg, dockerContext, platform})
	// must create a tar stream that looks somewhat normal to pass to stdout
	// what we need:
	// a config blob (random data)
	// a layer blob (random data)
	// a manifest blob (from the above)
	// an index blob (points to the manifest)
	// index.json (points to the index)
	tw := tar.NewWriter(stdout)
	defer tw.Close()
	buf := make([]byte, 128)

	var (
		configHash, layerHash, manifestHash, indexHash v1.Hash
		configSize, layerSize, manifestSize, indexSize int64
	)
	// config blob
	if _, err := rand.Read(buf); err != nil {
		return err
	}
	hash, _, err := v1.SHA256(bytes.NewReader(buf))
	if err != nil {
		return err
	}
	if err := tw.WriteHeader(&tar.Header{Name: fmt.Sprintf("blobs/sha256/%s", hash.Hex), Size: int64(len(buf))}); err != nil {
		return err
	}
	if _, err := tw.Write(buf); err != nil {
		return err
	}
	configHash = hash
	configSize = int64(len(buf))

	// layer blob
	if _, err := rand.Read(buf); err != nil {
		return err
	}
	hash, _, err = v1.SHA256(bytes.NewReader(buf))
	if err != nil {
		return err
	}
	if err := tw.WriteHeader(&tar.Header{Name: fmt.Sprintf("blobs/sha256/%s", hash.Hex), Size: int64(len(buf))}); err != nil {
		return err
	}
	if _, err := tw.Write(buf); err != nil {
		return err
	}
	layerHash = hash
	layerSize = int64(len(buf))

	// manifest
	manifest := v1.Manifest{
		Config: v1.Descriptor{
			MediaType: types.OCIConfigJSON,
			Size:      configSize,
			Digest:    configHash,
		},
		Layers: []v1.Descriptor{
			{
				MediaType: types.OCILayer,
				Size:      layerSize,
				Digest:    layerHash,
			},
		},
	}
	b, err := json.Marshal(manifest)
	if err != nil {
		return err
	}
	hash, _, err = v1.SHA256(bytes.NewReader(b))
	if err != nil {
		return err
	}
	if err := tw.WriteHeader(&tar.Header{Name: fmt.Sprintf("blobs/sha256/%s", hash.Hex), Size: int64(len(b))}); err != nil {
		return err
	}
	if _, err := tw.Write(b); err != nil {
		return err
	}
	manifestHash = hash
	manifestSize = int64(len(b))

	// index
	index := v1.IndexManifest{
		MediaType: types.OCIImageIndex,
		Manifests: []v1.Descriptor{
			{
				MediaType: types.OCIManifestSchema1,
				Size:      manifestSize,
				Digest:    manifestHash,
			},
		},
		SchemaVersion: 2,
	}
	b, err = json.Marshal(index)
	if err != nil {
		return err
	}
	hash, _, err = v1.SHA256(bytes.NewReader(b))
	if err != nil {
		return err
	}
	if err := tw.WriteHeader(&tar.Header{Name: fmt.Sprintf("blobs/sha256/%s", hash.Hex), Size: int64(len(b))}); err != nil {
		return err
	}
	if _, err := tw.Write(b); err != nil {
		return err
	}
	indexHash = hash
	indexSize = int64(len(b))

	// index.json
	index = v1.IndexManifest{
		MediaType: types.OCIImageIndex,
		Manifests: []v1.Descriptor{
			{
				MediaType: types.OCIImageIndex,
				Size:      indexSize,
				Digest:    indexHash,
				Annotations: map[string]string{
					imagespec.AnnotationRefName: tag,
					images.AnnotationImageName:  tag,
				},
			},
		},
		SchemaVersion: 2,
	}
	b, err = json.Marshal(index)
	if err != nil {
		return err
	}
	if err := tw.WriteHeader(&tar.Header{Name: "index.json", Size: int64(len(b))}); err != nil {
		return err
	}
	if _, err := tw.Write(b); err != nil {
		return err
	}
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
		_, _ = rand.Read(b)
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
	_, _ = rand.Read(b)
	descs, err := c.imageWriteStream(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	if len(descs) != 1 {
		return nil, fmt.Errorf("expected 1 descriptor, got %d", len(descs))
	}
	return c.NewSource(ref, architecture, &descs[1]), nil
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

func (c *cacheMocker) ImageLoad(r io.Reader) ([]registry.Descriptor, error) {
	if !c.enableImageLoad {
		return nil, errors.New("ImageLoad disabled")
	}
	return c.imageWriteStream(r)
}

func (c *cacheMocker) imageWriteStream(r io.Reader) ([]registry.Descriptor, error) {
	var (
		image string
		size  int64
		hash  v1.Hash
	)

	tarBytes, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("error reading data: %v", err)
	}
	var (
		tr    = tar.NewReader(bytes.NewReader(tarBytes))
		index bytes.Buffer
	)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return nil, err
		}

		filename := header.Name
		switch {
		case filename == "index.json":
			// any errors should stop and get reported
			if _, err := io.Copy(&index, tr); err != nil {
				return nil, fmt.Errorf("error reading data for file %s : %v", filename, err)
			}
		case strings.HasPrefix(filename, "blobs/sha256/"):
			// must have a file named blob/sha256/<hash>
			parts := strings.Split(filename, "/")
			// if we had a file that is just the directory, ignore it
			if len(parts) != 3 {
				continue
			}
			hash, err := v1.NewHash(fmt.Sprintf("%s:%s", parts[1], parts[2]))
			if err != nil {
				// malformed file
				return nil, fmt.Errorf("invalid hash filename for %s: %v", filename, err)
			}
			b, err := io.ReadAll(tr)
			if err != nil {
				return nil, fmt.Errorf("error reading data for file %s : %v", filename, err)
			}
			c.assignHash(hash.String(), b)
		}
	}
	if index.Len() != 0 {
		im, err := v1.ParseIndexManifest(&index)
		if err != nil {
			return nil, fmt.Errorf("error reading index.json")
		}
		for _, desc := range im.Manifests {
			if imgName, ok := desc.Annotations[images.AnnotationImageName]; ok {
				image = imgName
				size = desc.Size
				hash = desc.Digest
				break
			}
		}
	}

	desc := registry.Descriptor{
		MediaType: types.OCIManifestSchema1,
		Size:      size,
		Digest:    hash,
		Annotations: map[string]string{
			imagespec.AnnotationRefName: image,
		},
	}
	c.appendImage(image, desc)
	return []registry.Descriptor{desc}, nil
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

func (c *cacheMocker) GetContent(hash v1.Hash) (io.ReadCloser, error) {
	content, ok := c.hashes[hash.String()]
	if !ok {
		return nil, fmt.Errorf("no content found for hash: %s", hash.String())
	}
	return io.NopCloser(bytes.NewReader(content)), nil
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
	_, _ = rand.Read(b)
	return io.NopCloser(bytes.NewReader(b)), nil
}
func (c cacheMockerSource) Descriptor() *registry.Descriptor {
	return c.descriptor
}
func (c cacheMockerSource) SBoMs() ([]io.ReadCloser, error) {
	return nil, nil
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
			tt.p.dockerfile = "testdata/Dockerfile"
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
