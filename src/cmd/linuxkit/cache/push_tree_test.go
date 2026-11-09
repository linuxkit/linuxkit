package cache

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	ggcrlog "github.com/google/go-containerregistry/pkg/logs"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/util"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"golang.org/x/term"
)

func TestPushTreePreservesIndexManifestBlobHierarchy(t *testing.T) {
	f, o := completionFixture(t)
	rows, _ := o.treeSnapshot()
	require.Equal(t, f.root.Digest, rows[0].digest)
	require.Equal(t, "index", rows[0].kind)
	require.Empty(t, rows[0].prefix)
	require.Equal(t, f.nested.Digest, rows[1].digest)
	require.Equal(t, "index", rows[1].kind)
	require.Equal(t, "|-- ", rows[1].prefix)
	require.Equal(t, f.image.Digest, rows[2].digest)
	require.Equal(t, "manifest", rows[2].kind)
	require.Equal(t, "|   |-- ", rows[2].prefix)
	require.Equal(t, f.config.Digest, rows[3].digest)
	require.Empty(t, rows[3].kind)
	require.Equal(t, "|   |   |-- ", rows[3].prefix)
	require.Equal(t, f.layer.Digest, rows[4].digest)
	require.Equal(t, "|   |   `-- ", rows[4].prefix)
	require.Equal(t, []int{0, 1, 2}, rows[4].ancestors)
	require.Equal(t, f.attestation.Digest, rows[5].digest)
	require.Equal(t, "|   `-- ", rows[5].prefix)
	// The fixture contains a direct image reference and a repeated nested index.
	require.True(t, rows[8].shared)
	require.True(t, rows[9].shared)
	require.Equal(t, "`-- ", rows[11].prefix)
}

func TestPushTreeCountsSharedBlobsOnce(t *testing.T) {
	f, o := completionFixture(t)
	o.start(f.layer.Digest, f.layer.Size)
	o.read(f.layer.Digest, 1, 4, false)
	rows, _ := o.treeSnapshot()
	complete, total := pushTreeTotals(rows)
	expected := f.config.Size + f.layer.Size + f.attestationConfig.Size + f.attestationLayer.Size + f.root.Size + f.nested.Size + f.image.Size + f.attestation.Size
	require.EqualValues(t, 4, complete)
	require.Equal(t, expected, total)
	require.Equal(t, f.root.Size, rows[0].size)
	require.Zero(t, rows[0].read)
	require.Equal(t, "queued", rows[0].phase)
	require.Equal(t, f.image.Size, rows[2].size)
	var copies int
	for _, row := range rows {
		if row.digest == f.layer.Digest {
			copies++
			require.EqualValues(t, 4, row.read)
		}
	}
	require.Greater(t, copies, 1)
	require.True(t, o.existingManifest(f.attestation.Digest))
	rows, _ = o.treeSnapshot()
	require.Equal(t, "exists", rows[5].phase)
	o.publishedManifest(f.root.Digest)
	rows, _ = o.treeSnapshot()
	require.Equal(t, "sent", rows[0].phase)
	complete, total = pushTreeTotals(rows)
	require.Equal(t, total, complete)
}

func TestPushTreeColumnsAlignAtEveryDepth(t *testing.T) {
	_, o := completionFixture(t)
	rows, _ := o.treeSnapshot()
	now := time.Now()
	frame := renderPushFrame("tree upload", pushProgress{started: now, lastChange: now}, rows, 0, 120, 60)
	lines := strings.Split(strings.TrimSuffix(frame, "\n"), "\n")
	require.Len(t, lines, len(rows)+3, "two header lines and one shared-content note")
	column := -1
	for _, line := range lines[2 : 2+len(rows)] {
		require.LessOrEqual(t, len(line), 119)
		position := strings.Index(line, "[")
		require.GreaterOrEqual(t, position, 0)
		if column < 0 {
			column = position
		} else {
			require.Equal(t, column, position, line)
		}
	}
	require.Contains(t, frame, "index")
	require.Contains(t, frame, "manifest")
	require.Contains(t, frame, "* shared digest")
}

func TestSmallPushTreeRetainsActiveAncestors(t *testing.T) {
	f, o := completionFixture(t)
	o.start(f.attestationLayer.Digest, f.attestationLayer.Size)
	rows, _ := o.treeSnapshot()
	visible, hidden := visiblePushTree(rows, 4)
	require.Equal(t, len(rows)-4, hidden)
	require.Len(t, visible, 4)
	require.Equal(t, f.root.Digest, visible[0].digest)
	require.Equal(t, f.nested.Digest, visible[1].digest)
	require.Equal(t, f.attestation.Digest, visible[2].digest)
	require.Equal(t, f.attestationLayer.Digest, visible[3].digest)
	now := time.Now()
	frame := renderPushFrame("tree upload", pushProgress{started: now, lastChange: now}, rows, 0, 80, 8)
	require.LessOrEqual(t, strings.Count(frame, "\n"), 7)
	for _, line := range strings.Split(strings.TrimSuffix(frame, "\n"), "\n") {
		require.LessOrEqual(t, len(line), 79)
	}
	// Active metadata publication also takes priority over completed siblings.
	o.committed(f.attestationLayer.Digest)
	o.manifestPhase(f.attestation.Digest, "committing")
	rows, _ = o.treeSnapshot()
	visible, _ = visiblePushTree(rows, 3)
	require.Equal(t, f.attestation.Digest, visible[2].digest)
}

func TestPushTreeCompactColumnsAlign(t *testing.T) {
	_, o := completionFixture(t)
	rows, _ := o.treeSnapshot()
	now := time.Now()
	frame := renderPushFrame("tree upload", pushProgress{started: now, lastChange: now}, rows, 0, 60, 60)
	column := -1
	for _, line := range strings.Split(strings.TrimSuffix(frame, "\n"), "\n")[2 : 2+len(rows)] {
		require.LessOrEqual(t, len(line), 59)
		position := strings.Index(line, "/")
		require.GreaterOrEqual(t, position, 0)
		if column < 0 {
			column = position
		} else {
			require.Equal(t, column, position)
		}
	}
}

func TestPushTreeCycleIsBounded(t *testing.T) {
	o := newPushObserver()
	digest := v1.Hash{Algorithm: "sha256", Hex: strings.Repeat("a", 64)}
	o.rememberManifest(digest, []v1.Hash{digest})
	o.roots = []v1.Hash{digest}
	rows, _ := o.treeSnapshot()
	require.Len(t, rows, 2)
	require.Equal(t, "cycle", rows[0].phase)
	require.Equal(t, "cycle", rows[1].phase)
}

func TestObserveManifestPublicationWithoutConsumingBody(t *testing.T) {
	f, o := completionFixture(t)
	raw, err := f.p.cache.Bytes(f.root.Digest)
	require.NoError(t, err)
	request, err := http.NewRequest(http.MethodPut, "http://example.test/v2/test/manifests/tag", bytes.NewReader(raw))
	require.NoError(t, err)
	tr := pushTransport{observer: o, RoundTripper: completionRoundTripper(func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		require.Equal(t, raw, body)
		require.Equal(t, "waiting for registry", o.nodes[f.root.Digest].phase)
		return &http.Response{StatusCode: 201, Header: http.Header{}, Request: req}, nil
	})}
	_, err = tr.RoundTrip(request)
	require.NoError(t, err)
	rows, _ := o.treeSnapshot()
	require.Equal(t, "sent", rows[0].phase)
}

// Run from the compiled test binary in a PTY; ordinary go test uses pipes.
func TestPushTreeOnTerminal(t *testing.T) {
	if !term.IsTerminal(int(os.Stderr.Fd())) {
		t.Skip("requires a terminal")
	}
	t.Setenv("TERM", "xterm")
	logger := log.StandardLogger()
	out, formatter, level := logger.Out, logger.Formatter, logger.GetLevel()
	progress := ggcrlog.Progress
	defer func() {
		logger.SetOutput(out)
		logger.SetFormatter(formatter)
		logger.SetLevel(level)
		ggcrlog.Progress = progress
	}()
	require.NoError(t, util.SetupLogging(false, 1, false))
	logger.SetOutput(os.Stderr)
	f, o := completionFixture(t)
	d := newPushDisplay(o, "tree upload")
	defer d.Close()
	require.True(t, d.interactive())
	o.start(f.layer.Digest, f.layer.Size)
	o.read(f.layer.Digest, 1, 4, false)
	now := time.Now()
	state := pushProgress{started: now, lastChange: now}
	d.Report(state)
	o.publishedManifest(f.root.Digest)
	d.Finish(state, true)
}

func TestPushStatusKeepsFullIdleDuration(t *testing.T) {
	now := time.Now()
	state := pushProgress{started: now.Add(-time.Minute), lastChange: now.Add(-10 * time.Second)}
	rows := []blobRow{
		{digest: v1.Hash{Algorithm: "sha256", Hex: strings.Repeat("a", 64)}, kind: "index", blobUpload: blobUpload{phase: "sending"}},
		{digest: v1.Hash{Algorithm: "sha256", Hex: strings.Repeat("b", 64)}, prefix: "|-- ", kind: "manifest", ancestors: []int{0}, blobUpload: blobUpload{phase: "sending"}},
		{digest: v1.Hash{Algorithm: "sha256", Hex: strings.Repeat("c", 64)}, prefix: "|   `-- ", ancestors: []int{0, 1}, blobUpload: blobUpload{phase: "sending", size: 100 << 20, read: 12 << 20, attempt: 1, started: now.Add(-time.Minute), lastRead: now.Add(-10 * time.Second)}},
	}
	for _, width := range []int{40, 60, 80, 100, 120} {
		frame := renderPushFrame("upload", state, rows, 0, width, 24)
		require.Contains(t, frame, "send #1 idle 10s", fmt.Sprint(width))
		for _, line := range strings.Split(strings.TrimSuffix(frame, "\n"), "\n") {
			require.LessOrEqual(t, len(line), width-1)
		}
	}
	rows[2].lastRead = now.Add(-time.Hour - 2*time.Minute - 3*time.Second)
	require.Contains(t, renderPushFrame("upload", state, rows, 0, 80, 24), "send #1 idle 1h2m3s")
}
