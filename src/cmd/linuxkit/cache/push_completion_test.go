package cache

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	namepkg "github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/util"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func completionFixture(t *testing.T) (pushFixture, *pushObserver) {
	t.Helper()
	f := newPushFixture(t)
	root, err := f.p.FindRoot("example.test/kernel:test")
	require.NoError(t, err)
	index, err := root.ImageIndex()
	require.NoError(t, err)
	o := newPushObserver()
	require.NoError(t, o.discover(index))
	return f, o
}
func rowsByDigest(o *pushObserver) map[v1.Hash]blobUpload {
	rows, _ := o.snapshot()
	result := map[v1.Hash]blobUpload{}
	for _, row := range rows {
		result[row.digest] = row.blobUpload
	}
	return result
}
func TestExistingManifestMarksAllDescendants(t *testing.T) {
	f, o := completionFixture(t)
	require.True(t, o.existingManifest(f.attestation.Digest))
	rows := rowsByDigest(o)
	for _, d := range []v1.Descriptor{f.attestationConfig, f.attestationLayer} {
		require.Equal(t, "exists", rows[d.Digest].phase)
		require.Equal(t, d.Size, rows[d.Digest].read)
	}
	require.Equal(t, "queued", rows[f.layer.Digest].phase)
	// The nested index reaches both image and attestation, with shared references.
	require.True(t, o.existingManifest(f.root.Digest))
	for _, row := range rowsByDigest(o) {
		require.Equal(t, "exists", row.phase)
		require.Equal(t, row.size, row.read)
	}
	require.Empty(t, o.pending())
	require.Zero(t, o.traffic, "existing content is not newly transferred bytes")
}
func TestManifestCompletionPreservesSentBlobs(t *testing.T) {
	f, o := completionFixture(t)
	o.start(f.layer.Digest, f.layer.Size)
	o.read(f.layer.Digest, 1, 10, false)
	o.committed(f.layer.Digest)
	require.True(t, o.existingManifest(f.root.Digest))
	require.Equal(t, "sent", rowsByDigest(o)[f.layer.Digest].phase)
}
func TestCompletionMessagesRequireCompleteCoverage(t *testing.T) {
	f, o := completionFixture(t)
	unknown := v1.Hash{Algorithm: "sha256", Hex: strings.Repeat("f", 64)}
	blobEvents := []string{"existing blob: ", "pushed blob: ", "mounted blob: "}
	for _, prefix := range blobEvents {
		require.False(t, o.redundantCompletion(prefix+f.layer.Digest.String()))
	}
	require.False(t, o.redundantCompletion("existing manifest: "+f.root.Digest.String()))
	require.True(t, o.existingManifest(f.root.Digest))
	for _, prefix := range blobEvents {
		require.True(t, o.redundantCompletion(prefix+f.layer.Digest.String()))
		require.False(t, o.redundantCompletion(prefix+unknown.String()))
	}
	for _, message := range []string{
		"existing manifest: " + f.root.Digest.String(),
		"existing manifest: tag@" + f.root.Digest.String(),
		fmt.Sprintf("example.test/kernel:tag: digest: %s size: %d", f.root.Digest, f.root.Size),
	} {
		require.True(t, o.redundantCompletion(message), message)
	}
	for _, message := range []string{"ordinary message", "existing blob: malformed", "retrying an upload", "fallback tag foo already had referrer"} {
		require.False(t, o.redundantCompletion(message))
	}
	// Missing descendants disable suppression even for a known manifest.
	o.mu.Lock()
	delete(o.blobs, f.attestationLayer.Digest)
	o.mu.Unlock()
	require.False(t, o.existingManifest(f.root.Digest))
	require.False(t, o.redundantCompletion("existing manifest: "+f.root.Digest.String()))
}
func TestIncompleteOrCyclicGraphsRemainVisible(t *testing.T) {
	_, o := completionFixture(t)
	digest := v1.Hash{Algorithm: "sha256", Hex: strings.Repeat("f", 64)}
	o.rememberManifest(digest, []v1.Hash{digest})
	require.False(t, o.existingManifest(digest))
	require.False(t, o.redundantCompletion("existing manifest: "+digest.String()))
}
func TestCompletionFormatterOnlyFiltersGGCRInfo(t *testing.T) {
	f, o := completionFixture(t)
	require.True(t, o.existingManifest(f.root.Digest))
	formatter := &pushCompletionFormatter{Formatter: &log.TextFormatter{DisableTimestamp: true}, observer: o}
	for _, test := range []struct {
		level   log.Level
		source  bool
		message string
		hidden  bool
	}{
		{log.InfoLevel, true, "existing blob: " + f.layer.Digest.String(), true},
		{log.InfoLevel, true, "existing manifest: " + f.root.Digest.String(), true},
		{log.InfoLevel, false, "existing blob: " + f.layer.Digest.String(), false},
		{log.WarnLevel, true, "existing blob: " + f.layer.Digest.String(), false},
		{log.InfoLevel, true, "ordinary diagnostic", false},
		{log.InfoLevel, true, "pushed blob: sha256:" + strings.Repeat("f", 64), false},
	} {
		entry := log.NewEntry(log.StandardLogger())
		entry.Level = test.level
		entry.Message = test.message
		entry.Data[util.GGCRProgressField] = test.source
		output, err := formatter.Format(entry)
		require.NoError(t, err)
		require.Equal(t, test.hidden, len(output) == 0, test.message)
	}
}

type completionRoundTripper func(*http.Request) (*http.Response, error)

func (f completionRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
func TestObserveManifestHEADAndMount(t *testing.T) {
	for _, test := range []struct {
		status             int
		validDigest, byTag bool
		want               bool
	}{{200, true, false, true}, {200, true, true, true}, {404, true, false, false}, {200, false, false, false}} {
		f, o := completionFixture(t)
		identifier := f.root.Digest.String()
		if test.byTag {
			identifier = "tag"
		}
		request, err := http.NewRequest(http.MethodHead, "http://example.test/v2/test/manifests/"+identifier, nil)
		require.NoError(t, err)
		tr := pushTransport{observer: o, RoundTripper: completionRoundTripper(func(req *http.Request) (*http.Response, error) {
			header := http.Header{}
			header.Set("Content-Type", "application/vnd.oci.image.index.v1+json")
			value := f.root.Digest.String()
			if !test.validDigest {
				value = "sha256:" + strings.Repeat("f", 64)
			}
			header.Set("Docker-Content-Digest", value)
			return &http.Response{StatusCode: test.status, Header: header, ContentLength: f.root.Size, Request: req}, nil
		})}
		_, err = tr.RoundTrip(request)
		require.NoError(t, err)
		for _, row := range rowsByDigest(o) {
			require.Equal(t, test.want, row.phase == "exists")
		}
	}
	f, o := completionFixture(t)
	request, err := http.NewRequest(http.MethodPost, "http://example.test/v2/test/blobs/uploads/?mount="+f.layer.Digest.String(), nil)
	require.NoError(t, err)
	tr := pushTransport{observer: o, RoundTripper: completionRoundTripper(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 201, Header: http.Header{}, Request: req}, nil
	})}
	_, err = tr.RoundTrip(request)
	require.NoError(t, err)
	require.Equal(t, "mounted", rowsByDigest(o)[f.layer.Digest].phase)
	require.True(t, o.redundantCompletion("mounted blob: "+f.layer.Digest.String()))
}
func TestGGCRExistingManifestUpdatesRowsBeforeFinish(t *testing.T) {
	f, o := completionFixture(t)
	registry, server := newProgressRegistry(t)
	raw, err := f.p.cache.Bytes(f.attestation.Digest)
	require.NoError(t, err)
	registry.mu.Lock()
	registry.manifests[f.attestation.Digest.String()] = raw
	registry.mu.Unlock()
	root, err := f.p.FindRoot("example.test/kernel:test")
	require.NoError(t, err)
	index, err := root.ImageIndex()
	require.NoError(t, err)
	ref, err := namepkg.ParseReference(strings.TrimPrefix(server.URL, "http://") + "/test:tag")
	require.NoError(t, err)
	base := remote.DefaultTransport.(*http.Transport).Clone()
	defer base.CloseIdleConnections()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, remote.WriteIndex(ref, observedIndex{pushImageIndex: index, observer: o}, remote.WithContext(ctx), remote.WithTransport(pushTransport{RoundTripper: base, observer: o})))
	// No display Finish/skipped hook ran. GGCR skipped these two descendants,
	// and the HEAD observer already marked them as existing.
	rows := rowsByDigest(o)
	for _, d := range []v1.Descriptor{f.attestationConfig, f.attestationLayer} {
		require.Equal(t, "exists", rows[d.Digest].phase)
		registry.mu.Lock()
		_, uploaded := registry.blobs[d.Digest.String()]
		registry.mu.Unlock()
		require.False(t, uploaded)
	}
	require.Equal(t, "sent", rows[f.layer.Digest].phase)
}
