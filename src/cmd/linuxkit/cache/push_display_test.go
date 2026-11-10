package cache

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	ggcrlog "github.com/google/go-containerregistry/pkg/logs"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/util"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"golang.org/x/term"
)

func TestDiscoverPushBlobsBeforeUpload(t *testing.T) {
	f := newPushFixture(t)
	root, err := f.p.FindRoot("example.test/kernel:test")
	require.NoError(t, err)
	index, err := root.ImageIndex()
	require.NoError(t, err)
	observer := newPushObserver()
	require.NoError(t, observer.discover(index))
	rows, rate := observer.snapshot()
	require.Len(t, rows, 4, "shared image/attestation configs and layers should each appear once")
	require.Zero(t, rate)
	for _, row := range rows {
		require.Equal(t, "queued", row.phase)
		require.Zero(t, row.read)
		require.Positive(t, row.size)
	}
	observer.phase(f.layer.Digest, "checking")
	observer.committed(f.layer.Digest)
	rows, _ = observer.snapshot()
	var existing bool
	for _, row := range rows {
		if row.digest == f.layer.Digest {
			existing = true
			require.Equal(t, "exists", row.phase)
			require.Equal(t, row.size, row.read)
		}
	}
	require.True(t, existing)
}

func TestPushDisplayRetainsCompletedRows(t *testing.T) {
	o := newPushObserver()
	digest := v1.Hash{Algorithm: "sha256", Hex: strings.Repeat("a", 64)}
	o.start(digest, 100)
	o.read(digest, 1, 100, true)
	o.phase(digest, "committing")
	o.committed(digest)
	require.Empty(t, o.pending())
	rows, _ := o.snapshot()
	require.Len(t, rows, 1)
	require.Equal(t, "sent", rows[0].phase)
	o.start(digest, 100)
	o.read(digest, 2, 20, false)
	rows, _ = o.snapshot()
	require.EqualValues(t, 20, rows[0].read)
	require.Equal(t, 2, rows[0].attempt)
}

func TestRenderPushFrame(t *testing.T) {
	now := time.Now()
	state := pushProgress{complete: 8 << 20, total: 100 << 20, started: now.Add(-10 * time.Second), lastChange: now.Add(-2 * time.Second)}
	rows := []blobRow{{digest: v1.Hash{Algorithm: "sha256", Hex: strings.Repeat("a", 64)}, blobUpload: blobUpload{read: 8 << 20, size: 100 << 20, attempt: 1, phase: "sending", started: now.Add(-10 * time.Second), lastRead: now.Add(-2 * time.Second)}}}
	frame := renderPushFrame("test upload", state, rows, 800<<10, 100, 24)
	require.Contains(t, frame, "aaaaaaaaaaaa")
	require.Contains(t, frame, "8.0MiB/100.0MiB")
	require.Contains(t, frame, "800.0KiB/s")
	require.Contains(t, frame, "ETA 1m58s")
	require.Contains(t, frame, "send #1")
	require.Contains(t, frame, "[")
	for _, line := range strings.Split(strings.TrimSuffix(frame, "\n"), "\n") {
		require.LessOrEqual(t, len(line), 99)
	}
}

func TestRenderPushFrameFitsSmallTerminal(t *testing.T) {
	now := time.Now()
	state := pushProgress{started: now, lastChange: now}
	var rows []blobRow
	for n := 0; n < 30; n++ {
		phase := "sent"
		if n == 29 {
			phase = "sending"
		}
		rows = append(rows, blobRow{digest: v1.Hash{Algorithm: "sha256", Hex: fmt.Sprintf("%064x", n)}, blobUpload: blobUpload{phase: phase}})
	}
	frame := renderPushFrame("test upload", state, rows, 0, 40, 10)
	require.LessOrEqual(t, strings.Count(frame, "\n"), 9)
	require.Contains(t, frame, "more tree rows")
	for _, line := range strings.Split(strings.TrimSuffix(frame, "\n"), "\n") {
		require.LessOrEqual(t, len(line), 39)
	}
	// The unfinished row must be selected before completed rows, even though
	// the short digest is identical for these deliberately tiny test hashes.
	copy := append([]blobRow(nil), rows...)
	copy[29].digest.Hex = strings.Repeat("f", 64)
	require.Contains(t, renderPushFrame("test upload", state, copy, 0, 80, 10), "ffffffffffff")
}

func TestPushTerminalRedrawAndLogging(t *testing.T) {
	var output bytes.Buffer
	terminal := &pushTerminal{out: &output}
	terminal.Render("blob one\nblob two\n")
	_, err := terminal.Write([]byte("a diagnostic\n"))
	require.NoError(t, err)
	terminal.Render("blob one done\nblob two done\n")
	terminal.Close()
	_, err = terminal.Write([]byte("finished\n"))
	require.NoError(t, err)
	text := output.String()
	require.Contains(t, text, "\x1b[2A\r\x1b[Ja diagnostic\nblob one\nblob two\n")
	require.True(t, strings.HasSuffix(text, "blob one done\nblob two done\nfinished\n"))
}

func TestPushTerminalConcurrentLogging(t *testing.T) {
	var output bytes.Buffer
	terminal := &pushTerminal{out: &output}
	var group sync.WaitGroup
	for n := 0; n < 10; n++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for i := 0; i < 50; i++ {
				terminal.Render("blob progress\n")
				_, _ = terminal.Write([]byte("message\n"))
			}
		}()
	}
	group.Wait()
	terminal.Close()
	require.Equal(t, 500, strings.Count(output.String(), "message\n"))
}

func TestPlainPushStatusIsCompact(t *testing.T) {
	now := time.Now()
	observer := newPushObserver()
	digest := v1.Hash{Algorithm: "sha256", Hex: strings.Repeat("a", 64)}
	observer.start(digest, 100)
	message := plainPushStatus("upload", pushProgress{started: now, lastChange: now}, observer)
	require.Contains(t, message, "1 pending")
	require.Contains(t, message, "ETA --")
	require.NotContains(t, message, digest.String())
	require.NotContains(t, message, "\x1b")
}

// Also runnable directly from a compiled test binary under a terminal. Normal
// go test captures output through pipes, so it skips only this integration check.
func TestPushDisplayOnTerminal(t *testing.T) {
	if !term.IsTerminal(int(os.Stderr.Fd())) {
		t.Skip("requires terminal; run the compiled test binary in a PTY")
	}
	t.Setenv("TERM", "xterm")
	logger := log.StandardLogger()
	original, level, originalFormatter := logger.Out, logger.GetLevel(), logger.Formatter
	progress := ggcrlog.Progress
	require.NoError(t, util.SetupLogging(false, 1, false))
	activeFormatter := logger.Formatter
	logger.SetOutput(os.Stderr)
	defer func() {
		logger.SetOutput(original)
		logger.SetLevel(level)
		logger.SetFormatter(originalFormatter)
		ggcrlog.Progress = progress
	}()
	o := newPushObserver()
	d := newPushDisplay(o, "terminal upload")
	defer d.Close()
	require.True(t, d.interactive())
	require.ErrorContains(t, (PushOptions{TraceUpload: true}).Validate(), "--progress=plain")
	require.NoError(t, (PushOptions{TraceUpload: true, Progress: PushProgressPlain}).Validate())
	require.NoError(t, (PushOptions{}).Validate())
	digest := v1.Hash{Algorithm: "sha256", Hex: strings.Repeat("a", 64)}
	o.start(digest, 100)
	o.read(digest, 1, 50, false)
	now := time.Now()
	state := pushProgress{complete: 50, total: 100, started: now, lastChange: now}
	d.Report(state)
	log.Warn("diagnostic stays visible")
	o.committed(digest)
	ggcrlog.Progress.Print("pushed blob: " + digest.String())
	ggcrlog.Progress.Print("existing blob: sha256:" + strings.Repeat("b", 64))
	state.complete = 100
	d.Finish(state, true)
	require.Same(t, os.Stderr, logger.Out)
	require.Same(t, activeFormatter, logger.Formatter)
	// Explicit plain mode must stay plain even on this real terminal.
	plainObserver := newPushObserver()
	plainObserver.progressMode = PushProgressPlain
	plain := newPushDisplay(plainObserver, "forced plain")
	defer plain.Close()
	require.False(t, plain.interactive())
	require.Same(t, os.Stderr, logger.Out)
	ggcrlog.Progress.Print("existing blob: sha256:" + strings.Repeat("c", 64))
}

func TestPlainDisplayKeepsCompletionMessages(t *testing.T) {
	f, o := completionFixture(t)
	require.True(t, o.existingManifest(f.root.Digest))
	logger := log.StandardLogger()
	out, formatter, level := logger.Out, logger.Formatter, logger.GetLevel()
	defer func() { logger.SetOutput(out); logger.SetFormatter(formatter); logger.SetLevel(level) }()
	var output bytes.Buffer
	plainFormatter := &log.TextFormatter{DisableTimestamp: true}
	logger.SetOutput(&output)
	logger.SetFormatter(plainFormatter)
	logger.SetLevel(log.InfoLevel)
	d := newPushDisplay(o, "plain push")
	defer d.Close()
	require.False(t, d.interactive())
	require.Same(t, plainFormatter, logger.Formatter)
	message := "existing blob: " + f.layer.Digest.String()
	log.WithField(util.GGCRProgressField, true).Info(message)
	require.Contains(t, output.String(), message)
}

func TestEstimatePushRemaining(t *testing.T) {
	now := time.Now()
	state := pushProgress{started: now.Add(-time.Minute), lastChange: now}
	row := func(digest string, size, read int64, phase string) blobRow {
		return blobRow{digest: v1.Hash{Algorithm: "sha256", Hex: digest}, blobUpload: blobUpload{size: size, read: read, phase: phase}}
	}
	active := row("active", 1000, 250, "sending")
	existing := row("existing", 5000, 5000, "exists")
	cases := []struct {
		name string
		rows []blobRow
		rate float64
		want string
	}{
		{"transfer", []blobRow{active}, 100, "8s"}, // Round up fractional seconds.
		{"faster", []blobRow{active}, 250, "3s"},
		{"existing and shared", []blobRow{existing, active, active}, 100, "8s"},
		{"queued included", []blobRow{active, row("queued", 1000, 0, "queued")}, 100, "18s"},
		{"retry resets progress", []blobRow{row("active", 1000, 0, "sending")}, 100, "10s"},
		{"no observations", nil, 100, "--"},
		{"no rate", []blobRow{active}, 0, "--"},
		{"negative rate", []blobRow{active}, -1, "--"},
		{"invalid rate", []blobRow{active}, math.NaN(), "--"},
		{"infinite rate", []blobRow{active}, math.Inf(1), "--"},
		{"duration overflow", []blobRow{active}, math.SmallestNonzeroFloat64, "--"},
		{"awaiting acknowledgement", []blobRow{row("active", 1000, 1000, "waiting for registry")}, 100, "waiting"},
		{"zero-byte pending", []blobRow{row("empty", 0, 0, "queued")}, 0, "waiting"},
		{"complete", []blobRow{existing}, 0, "0s"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, estimatePushRemaining(tc.rows, state, tc.rate, now))
		})
	}
	stalled := state
	stalled.lastChange = now.Add(-30 * time.Second)
	require.Equal(t, "--", estimatePushRemaining([]blobRow{active}, stalled, 100, now))
	failed := state
	failed.finished = now
	require.Equal(t, "--", estimatePushRemaining([]blobRow{active}, failed, 100, now))
}

func TestPushETAVisibleInNormalTerminal(t *testing.T) {
	now := time.Now()
	rows := []blobRow{{digest: v1.Hash{Algorithm: "sha256", Hex: "abc"}, blobUpload: blobUpload{size: 1000, read: 250, phase: "sending"}}}
	frame := renderPushFrame("upload", pushProgress{started: now.Add(-time.Minute), lastChange: now}, rows, 100, 80, 24)
	require.Contains(t, strings.Split(frame, "\n")[1], "ETA 8s")
	for _, line := range strings.Split(strings.TrimSuffix(frame, "\n"), "\n") {
		require.LessOrEqual(t, len(line), 79)
	}
}
