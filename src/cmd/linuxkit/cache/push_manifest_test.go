package cache

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/require"
)

func TestManifestRowsShowOnlyTheirOwnBytes(t *testing.T) {
	f, o := completionFixture(t)
	o.start(f.layer.Digest, f.layer.Size)
	o.read(f.layer.Digest, 1, 4, false)
	rows, _ := o.treeSnapshot()
	require.Equal(t, f.root.Size, rows[0].size)
	require.Zero(t, rows[0].read)
	require.Equal(t, "queued", rows[0].phase)
	require.Equal(t, f.image.Size, rows[2].size)
	require.Zero(t, rows[2].read)
	require.Equal(t, "queued", rows[2].phase)
	raw, err := f.p.cache.Bytes(f.image.Digest)
	require.NoError(t, err)
	body := o.manifestBody(f.image.Digest, int64(len(raw)), io.NopCloser(bytes.NewReader(raw)))
	require.Equal(t, "sending", o.nodes[f.image.Digest].phase)
	buffer := make([]byte, 7)
	n, err := body.Read(buffer)
	require.NoError(t, err)
	require.Equal(t, 7, n)
	rows, _ = o.treeSnapshot()
	require.EqualValues(t, 7, rows[2].read)
	require.Equal(t, f.image.Size, rows[2].size)
	require.Zero(t, rows[0].read, "sending a manifest must not advance its parent index")
	_, err = io.Copy(io.Discard, body)
	require.NoError(t, err)
	require.NoError(t, body.Close())
	require.Equal(t, "waiting for registry", o.nodes[f.image.Digest].phase)
	require.Contains(t, o.pending(), f.image.Digest.String()+" (manifest)")
	require.Equal(t, f.image.Size, o.nodes[f.image.Digest].read)
	o.publishedManifest(f.image.Digest)
	require.Equal(t, "sent", o.nodes[f.image.Digest].phase)
	require.NotContains(t, o.pending(), f.image.Digest.String()+" (manifest)")
	require.Equal(t, f.image.Size, o.nodes[f.image.Digest].read)
}

func TestManifestBodyReplayResetsOwnProgress(t *testing.T) {
	f, o := completionFixture(t)
	raw, err := f.p.cache.Bytes(f.root.Digest)
	require.NoError(t, err)
	request, err := http.NewRequest(http.MethodPut, "http://example.test/v2/test/manifests/tag", bytes.NewReader(raw))
	require.NoError(t, err)
	tr := pushTransport{observer: o, RoundTripper: completionRoundTripper(func(req *http.Request) (*http.Response, error) {
		first := make([]byte, 10)
		_, err := io.ReadFull(req.Body, first)
		require.NoError(t, err)
		require.EqualValues(t, 10, o.nodes[f.root.Digest].read)
		replay, err := req.GetBody()
		require.NoError(t, err)
		require.Zero(t, o.nodes[f.root.Digest].read)
		require.Equal(t, 2, o.nodes[f.root.Digest].attempt)
		// A late close from the previous reader must not change the new attempt.
		require.NoError(t, req.Body.Close())
		require.Equal(t, "sending", o.nodes[f.root.Digest].phase)
		data, err := io.ReadAll(replay)
		require.NoError(t, err)
		require.Equal(t, raw, data)
		require.NoError(t, replay.Close())
		return &http.Response{StatusCode: 201, Header: http.Header{}, Request: req}, nil
	})}
	_, err = tr.RoundTrip(request)
	require.NoError(t, err)
	require.Equal(t, "sent", o.nodes[f.root.Digest].phase)
	require.Equal(t, f.root.Size, o.nodes[f.root.Digest].read)
	require.Equal(t, f.root.Size+10, o.traffic, "traffic includes replayed bytes, unlike the current-attempt counter")
}

func TestExistingMetadataShowsOwnSizeImmediately(t *testing.T) {
	f, o := completionFixture(t)
	require.True(t, o.existingManifest(f.root.Digest))
	rows, _ := o.treeSnapshot()
	complete, total := pushTreeTotals(rows)
	require.Equal(t, total, complete)
	for _, row := range rows {
		if row.digest == f.root.Digest {
			require.Equal(t, f.root.Size, row.size)
			require.Equal(t, f.root.Size, row.read)
		}
		if row.digest == f.image.Digest {
			require.Equal(t, f.image.Size, row.size)
			require.Equal(t, f.image.Size, row.read)
		}
	}
	require.Zero(t, o.traffic, "already present metadata is not new traffic")
}

func TestManifestProgressUpdatesIdleTimestamp(t *testing.T) {
	f, o := completionFixture(t)
	raw, err := f.p.cache.Bytes(f.root.Digest)
	require.NoError(t, err)
	before := time.Now().Add(-time.Minute)
	body := o.manifestBody(f.root.Digest, int64(len(raw)), io.NopCloser(bytes.NewReader(raw)))
	defer func() { require.NoError(t, body.Close()) }()
	_, err = body.Read(make([]byte, 5))
	require.NoError(t, err)
	state := o.progressState(pushProgress{started: before, lastChange: before})
	require.True(t, state.lastChange.After(before))
	require.EqualValues(t, 5, state.complete)
}

func TestCompletionCoverageIncludesMetadataRows(t *testing.T) {
	f, o := completionFixture(t)
	require.True(t, o.existingManifest(f.root.Digest))
	o.manifestPhase(f.root.Digest, "sending")
	require.False(t, o.redundantCompletion("existing manifest: "+f.root.Digest.String()))
	unknown := v1.Hash{Algorithm: "sha256", Hex: strings.Repeat("f", 64)}
	require.False(t, o.existingManifest(unknown))
}
