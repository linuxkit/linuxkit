package cache

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/stretchr/testify/require"
)

// Use small real OCI files so tests cover lazy layout reads and missing blobs.
type pushFixture struct {
	p                                                                                    *Provider
	root, nested, image, config, layer, attestation, attestationConfig, attestationLayer v1.Descriptor
}

func newPushFixture(t *testing.T) pushFixture {
	t.Helper()
	p, err := NewProvider(t.TempDir())
	require.NoError(t, err)
	blob := func(mediaType types.MediaType, raw []byte) v1.Descriptor {
		h, size, err := v1.SHA256(bytes.NewReader(raw))
		require.NoError(t, err)
		path := filepath.Join(p.dir, "blobs", h.Algorithm, h.Hex)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
		require.NoError(t, os.WriteFile(path, raw, 0644))
		return v1.Descriptor{MediaType: mediaType, Digest: h, Size: size}
	}
	jsonBlob := func(mediaType types.MediaType, value interface{}) v1.Descriptor {
		raw, err := json.Marshal(value)
		require.NoError(t, err)
		return blob(mediaType, raw)
	}
	f := pushFixture{p: p}
	f.config = blob(types.OCIConfigJSON, []byte(`{"architecture":"arm64","os":"linux"}`))
	f.layer = blob(types.OCILayer, []byte("test image layer"))
	f.image = jsonBlob(types.OCIManifestSchema1, v1.Manifest{SchemaVersion: 2, MediaType: types.OCIManifestSchema1, Config: f.config, Layers: []v1.Descriptor{f.layer}})
	f.image.Platform = &v1.Platform{OS: "linux", Architecture: "arm64"}
	f.attestationConfig = blob(types.OCIConfigJSON, []byte(`{"architecture":"unknown","os":"unknown"}`))
	f.attestationLayer = blob(types.MediaType("application/vnd.in-toto+json"), []byte(`{"predicateType":"https://spdx.dev/Document"}`))
	f.attestation = jsonBlob(types.OCIManifestSchema1, v1.Manifest{SchemaVersion: 2, MediaType: types.OCIManifestSchema1, Config: f.attestationConfig, Layers: []v1.Descriptor{f.attestationLayer}})
	f.attestation.Platform = &v1.Platform{OS: "unknown", Architecture: "unknown"}
	f.attestation.Annotations = map[string]string{"vnd.docker.reference.digest": f.image.Digest.String()}
	f.nested = jsonBlob(types.OCIImageIndex, v1.IndexManifest{SchemaVersion: 2, MediaType: types.OCIImageIndex, Manifests: []v1.Descriptor{f.image, f.attestation}})
	// The repeated nested descriptor verifies shared subtrees are deduplicated.
	f.root = jsonBlob(types.OCIImageIndex, v1.IndexManifest{SchemaVersion: 2, MediaType: types.OCIImageIndex, Manifests: []v1.Descriptor{f.nested, f.image, f.nested}})
	require.NoError(t, p.DescriptorWrite("example.test/kernel:test", f.root))
	return f
}

func (f pushFixture) path(d v1.Descriptor) string {
	return filepath.Join(f.p.dir, "blobs", d.Digest.Algorithm, d.Digest.Hex)
}

func TestValidatePushContentsComplete(t *testing.T) {
	f := newPushFixture(t)
	require.NoError(t, f.p.validatePushContents("example.test/kernel:test"))
}

func TestValidatePushContentsMissing(t *testing.T) {
	for _, part := range []string{"root", "nested", "image", "config", "layer", "attestation", "attestation config", "attestation layer"} {
		t.Run(part, func(t *testing.T) {
			f := newPushFixture(t)
			d := map[string]v1.Descriptor{"root": f.root, "nested": f.nested, "image": f.image, "config": f.config, "layer": f.layer, "attestation": f.attestation, "attestation config": f.attestationConfig, "attestation layer": f.attestationLayer}[part]
			require.NoError(t, os.Remove(f.path(d)))
			err := f.p.validatePushContents("example.test/kernel:test")
			require.Error(t, err)
			require.ErrorIs(t, err, os.ErrNotExist)
			var pathErr *os.PathError
			require.ErrorAs(t, err, &pathErr)
			require.Contains(t, err.Error(), d.Digest.String())
			require.Contains(t, err.Error(), f.path(d))
			require.Contains(t, err.Error(), "no upload attempted")
			if part == "image" {
				require.Contains(t, err.Error(), "platform linux/arm64")
			}
		})
	}
}

func TestPushReportsMissingSiblingsBeforeNetwork(t *testing.T) {
	f := newPushFixture(t)
	require.NoError(t, os.Remove(f.path(f.image)))
	require.NoError(t, os.Remove(f.path(f.attestationLayer)))
	// An invalid destination host must never be contacted: preflight fails first.
	err := f.p.Push("example.test/kernel:test", "127.0.0.1:1/kernel:test", false, true)
	var report *pushContentError
	require.ErrorAs(t, err, &report)
	require.Len(t, report.issues, 2)
	require.Contains(t, err.Error(), f.image.Digest.String())
	require.Contains(t, err.Error(), f.attestationLayer.Digest.String())
	require.Contains(t, err.Error(), "manifests[0]")
}

func TestValidatePushContentsInvalid(t *testing.T) {
	for _, kind := range []string{"size", "json", "directory"} {
		t.Run(kind, func(t *testing.T) {
			f := newPushFixture(t)
			switch kind {
			case "size":
				require.NoError(t, os.WriteFile(f.path(f.layer), []byte("short"), 0644))
			case "json":
				require.NoError(t, os.WriteFile(f.path(f.nested), []byte(strings.Repeat("!", int(f.nested.Size))), 0644))
			case "directory":
				require.NoError(t, os.Remove(f.path(f.layer)))
				require.NoError(t, os.Mkdir(f.path(f.layer), 0755))
			}
			err := f.p.validatePushContents("example.test/kernel:test")
			require.Error(t, err)
			require.False(t, errors.Is(err, os.ErrNotExist))
			want := map[string]string{"size": "size mismatch", "json": "malformed index JSON", "directory": "not a regular file"}[kind]
			require.Contains(t, err.Error(), want)
		})
	}
}

func TestFindRootPreservesReadError(t *testing.T) {
	f := newPushFixture(t)
	require.NoError(t, os.Remove(f.path(f.root)))
	_, err := f.p.FindRoot("example.test/kernel:test")
	require.ErrorIs(t, err, os.ErrNotExist)
	require.Contains(t, err.Error(), f.root.Digest.String())
	_, err = f.p.FindRoot("example.test/kernel:absent")
	require.ErrorContains(t, err, "is not present in cache index")
	require.False(t, errors.Is(err, os.ErrNotExist))
}

func TestValidatePushContentsSingleImage(t *testing.T) {
	f := newPushFixture(t)
	require.NoError(t, f.p.DescriptorWrite("example.test/kernel:single", f.image))
	require.NoError(t, f.p.validatePushContents("example.test/kernel:single"))
	require.NoError(t, os.Remove(f.path(f.layer)))
	require.ErrorIs(t, f.p.validatePushContents("example.test/kernel:single"), os.ErrNotExist)
}

func TestValidatePushContentsCycle(t *testing.T) {
	f := newPushFixture(t)
	// Fabricate a cycle without computing hashes: the lightweight preflight
	// deliberately does not verify content digests, but must still terminate.
	d := v1.Descriptor{MediaType: types.OCIImageIndex, Digest: v1.Hash{Algorithm: "sha256", Hex: strings.Repeat("0", 64)}}
	var raw []byte
	for {
		var err error
		raw, err = json.Marshal(v1.IndexManifest{SchemaVersion: 2, MediaType: types.OCIImageIndex, Manifests: []v1.Descriptor{d}})
		require.NoError(t, err)
		if d.Size == int64(len(raw)) {
			break
		}
		d.Size = int64(len(raw))
	}
	require.NoError(t, os.WriteFile(f.path(d), raw, 0644))
	require.NoError(t, f.p.DescriptorWrite("example.test/kernel:cycle", d))
	require.ErrorContains(t, f.p.validatePushContents("example.test/kernel:cycle"), "descriptor cycle detected")
}

func TestValidatePushContentsOpenError(t *testing.T) {
	f := newPushFixture(t)
	require.NoError(t, os.Chmod(f.path(f.layer), 0000))
	t.Cleanup(func() { _ = os.Chmod(f.path(f.layer), 0644) })
	if file, err := os.Open(f.path(f.layer)); err == nil {
		_ = file.Close()
		t.Skip("current user can read files without permission bits")
	}
	err := f.p.validatePushContents("example.test/kernel:test")
	require.ErrorIs(t, err, os.ErrPermission)
	require.Contains(t, err.Error(), "cannot open cache blob")
}

func TestValidatePushContentsTree(t *testing.T) {
	f := newPushFixture(t)
	require.NoError(t, os.Remove(f.path(f.attestationLayer)))
	err := f.p.validatePushContents("example.test/kernel:test")
	require.ErrorIs(t, err, os.ErrNotExist)
	output := err.Error()
	require.Contains(t, output, f.root.Digest.String()+" root index [incomplete]")
	require.Contains(t, output, "|-- "+f.nested.Digest.String()+" manifests[0] nested index [incomplete]")
	require.Contains(t, output, "|   |-- "+f.image.Digest.String()+" manifests[0] image manifest [complete] platform linux/arm64")
	require.Contains(t, output, "|   |   |-- "+f.config.Digest.String()+" config [complete]")
	require.Contains(t, output, "|   |   `-- "+f.layer.Digest.String()+" layers[0] [complete]")
	require.Contains(t, output, "|   `-- "+f.attestation.Digest.String()+" manifests[1] image manifest [incomplete] platform unknown/unknown")
	require.Contains(t, output, "|       `-- "+f.attestationLayer.Digest.String()+" layers[0] [MISSING]")
	require.Contains(t, output, "`-- "+f.nested.Digest.String()+" manifests[2] nested index [incomplete]")
	require.Contains(t, output, "shared subtree shown above")
	require.NotContains(t, output, "\x1b")
	require.NotContains(t, output, " -> ")
	require.NotContains(t, output, "NOT CHECKED")                                  // Missing leaf has no hidden descendants.
	require.Equal(t, 1, strings.Count(output, f.attestationLayer.Digest.String())) // Shared subtree is expanded once.
}

func TestValidatePushContentsMissingIndexTree(t *testing.T) {
	f := newPushFixture(t)
	require.NoError(t, os.Remove(f.path(f.nested)))
	err := f.p.validatePushContents("example.test/kernel:test")
	require.ErrorIs(t, err, os.ErrNotExist)
	require.Contains(t, err.Error(), f.nested.Digest.String()+" manifests[0] nested index [MISSING]")
	require.Contains(t, err.Error(), "descendants: NOT CHECKED (references unavailable)")
	// Independent siblings are still checked and displayed.
	require.Contains(t, err.Error(), f.image.Digest.String()+" manifests[1] image manifest [complete]")
	require.Contains(t, err.Error(), f.config.Digest.String()+" config [complete]")
	require.NotContains(t, err.Error(), f.attestation.Digest.String())
}
