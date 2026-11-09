package cache

import (
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	log "github.com/sirupsen/logrus"
	"golang.org/x/term"
)

// discover enumerates descriptors before transfer so queued blobs have rows too.
// It reads manifests, not compressed layer payloads. Shared content is deduplicated.
func (o *pushObserver) discover(root partial.Describable) error {
	seen := map[v1.Hash]bool{}
	var walk func(partial.Describable) error
	register := func(d v1.Descriptor) {
		o.mu.Lock()
		defer o.mu.Unlock()
		if _, ok := o.blobs[d.Digest]; !ok {
			o.blobs[d.Digest] = blobUpload{size: d.Size, phase: "queued"}
		}
	}
	walk = func(value partial.Describable) error {
		digest, err := value.Digest()
		if err != nil {
			return err
		}
		if seen[digest] {
			return nil
		}
		seen[digest] = true
		switch item := value.(type) {
		case v1.Image:
			manifest, err := item.Manifest()
			if err != nil {
				return err
			}
			register(manifest.Config)
			dependencies := []v1.Hash{manifest.Config.Digest}
			for _, layer := range manifest.Layers {
				register(layer)
				dependencies = append(dependencies, layer.Digest)
			}
			o.rememberManifest(digest, dependencies)
			size, err := item.Size()
			if err != nil {
				return err
			}
			o.describeManifest(digest, "manifest", size)
		case v1.ImageIndex:
			children, err := partial.Manifests(item)
			if err != nil {
				return err
			}
			var dependencies []v1.Hash
			for _, child := range children {
				if err := walk(child); err != nil {
					return err
				}
				childDigest, err := child.Digest()
				if err != nil {
					return err
				}
				dependencies = append(dependencies, childDigest)
			}
			o.rememberManifest(digest, dependencies)
			size, err := item.Size()
			if err != nil {
				return err
			}
			o.describeManifest(digest, "index", size)
		case v1.Layer:
			desc, err := partial.Descriptor(item)
			if err != nil {
				return err
			}
			register(*desc)
		}
		return nil
	}
	if err := walk(root); err != nil {
		return err
	}
	digest, err := root.Digest()
	if err != nil {
		return err
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	for _, existing := range o.roots {
		if existing == digest {
			return nil
		}
	}
	o.roots = append(o.roots, digest)
	return nil
}

type blobRow struct {
	digest v1.Hash
	blobUpload
	prefix    string
	kind      string // index/manifest for metadata objects; empty for blobs
	ancestors []int
	shared    bool
}

func (o *pushObserver) snapshot() ([]blobRow, float64) {
	o.mu.Lock()
	defer o.mu.Unlock()
	rows := make([]blobRow, 0, len(o.blobs))
	for digest, state := range o.blobs {
		rows = append(rows, blobRow{digest: digest, blobUpload: state})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].digest.String() < rows[j].digest.String() })
	seconds := time.Since(o.started).Seconds()
	rate := float64(0)
	if seconds > 0 {
		rate = float64(o.traffic) / seconds
	}
	return rows, rate
}
func (o *pushObserver) skipped() {
	o.mu.Lock()
	defer o.mu.Unlock()
	for digest, state := range o.blobs {
		if state.attempt == 0 && !completedBlob(state.phase) {
			state.phase = "skipped"
			state.read = state.size
			o.blobs[digest] = state
		}
	}
}
func plainPushStatus(label string, state pushProgress, observer *pushObserver) string {
	rows, rate := observer.treeSnapshot()
	state = observer.progressState(state)
	pending, done := 0, 0
	seen := map[v1.Hash]bool{}
	for _, row := range rows {
		if seen[row.digest] {
			continue
		}
		seen[row.digest] = true
		if completedBlob(row.phase) {
			done++
		} else {
			pending++
		}
	}
	return fmt.Sprintf("%s: %s; %s/s; ETA %s; %d pending, %d done", label, state.summary(), formatPushBytes(rate), estimatePushRemaining(rows, state, rate, time.Now()), pending, done)
}

// Only one interactive display can own the process logger at a time. A second
// concurrent push uses plain logs, which the owning display prints above its rows.
var pushDisplayMu sync.Mutex

type pushDisplay struct {
	observer          *pushObserver
	label             string
	terminal          *pushTerminal
	logger            *log.Logger
	original          io.Writer
	originalFormatter log.Formatter
	formatter         *pushCompletionFormatter
}

func newPushDisplay(observer *pushObserver, label string) *pushDisplay {
	d := &pushDisplay{observer: observer, label: label, logger: log.StandardLogger()}
	pushDisplayMu.Lock()
	defer pushDisplayMu.Unlock()
	output := d.logger.Out
	file, isFile := output.(*os.File)
	isTerminal := isFile && term.IsTerminal(int(file.Fd())) && os.Getenv("TERM") != "dumb"
	_, owned := output.(*pushTerminal)
	mode := observer.progressMode
	show := log.IsLevelEnabled(log.InfoLevel) && !owned && (mode == PushProgressTTY || ((mode == PushProgressAuto || mode == "") && isTerminal && log.GetLevel() == log.InfoLevel))
	if show {
		d.original = output
		d.terminal = &pushTerminal{out: output, dimensions: func() (int, int) {
			if !isFile {
				return 80, 24
			}
			w, h, err := term.GetSize(int(file.Fd()))
			if err != nil || w < 1 || h < 1 {
				return 80, 24
			}
			return w, h
		}}
		observer.mu.Lock()
		observer.interactive = true
		observer.mu.Unlock()
		d.originalFormatter = d.logger.Formatter
		d.formatter = &pushCompletionFormatter{Formatter: d.originalFormatter, observer: observer}
		d.logger.SetFormatter(d.formatter)
		d.logger.SetOutput(d.terminal)
		now := time.Now()
		d.Report(pushProgress{started: now, lastChange: now})
	}
	return d
}
func (d *pushDisplay) interactive() bool { return d.terminal != nil }
func (d *pushDisplay) Report(state pushProgress) {
	if d.terminal == nil {
		log.Infof("%s", plainPushStatus(d.label, state, d.observer))
		return
	}
	rows, rate := d.observer.treeSnapshot()
	state = d.observer.progressState(state)
	width, height := d.terminal.dimensions()
	d.terminal.Render(renderPushFrame(d.label, state, rows, rate, width, height))
}
func (d *pushDisplay) Finish(state pushProgress, success bool) {
	if success {
		d.observer.skipped()
	}
	if d.terminal != nil {
		d.Report(state)
	}
	d.Close()
}
func (d *pushDisplay) Close() {
	if d.terminal == nil {
		return
	}
	pushDisplayMu.Lock()
	defer pushDisplayMu.Unlock()
	d.terminal.Close()
	if d.logger.Out == d.terminal {
		d.logger.SetOutput(d.original)
	}
	if d.logger.Formatter == d.formatter {
		d.logger.SetFormatter(d.originalFormatter)
	}
}

// pushTerminal serializes redraws with ordinary log messages. It doesn't replace
// GGCR's global loggers; errors and messages remain visible above the table.
type pushTerminal struct {
	mu         sync.Mutex
	out        io.Writer
	dimensions func() (int, int)
	frame      string
	lines      int
	closed     bool
}

func (t *pushTerminal) clear() string {
	if t.lines == 0 {
		return ""
	}
	return fmt.Sprintf("\x1b[%dA\r\x1b[J", t.lines)
}
func (t *pushTerminal) Render(frame string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return
	}
	_, _ = io.WriteString(t.out, t.clear()+frame)
	t.frame = frame
	t.lines = strings.Count(frame, "\n")
}
func (t *pushTerminal) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return t.out.Write(p)
	}
	text := t.clear() + string(p)
	if len(p) > 0 && p[len(p)-1] != '\n' {
		text += "\n"
	}
	text += t.frame
	_, err := io.WriteString(t.out, text)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}
func (t *pushTerminal) Close() { t.mu.Lock(); defer t.mu.Unlock(); t.closed = true }

func formatPushBytes(n float64) string {
	if n < 0 {
		n = 0
	}
	for _, unit := range []struct {
		size float64
		name string
	}{{1 << 30, "GiB"}, {1 << 20, "MiB"}, {1 << 10, "KiB"}} {
		if n >= unit.size {
			return fmt.Sprintf("%.1f%s", n/unit.size, unit.name)
		}
	}
	return fmt.Sprintf("%.0fB", n)
}

// Estimate the remaining client-side transfer using the same aggregate rate
// shown in the summary. Existing content and shared digests count only once;
// retries reset their artifact's read count and thus increase remaining work.
// Registry acknowledgement time cannot be inferred from byte throughput.
func estimatePushRemaining(rows []blobRow, state pushProgress, rate float64, now time.Time) string {
	if len(rows) == 0 {
		return "--"
	}
	done := true
	for _, row := range rows {
		done = done && completedBlob(row.phase)
	}
	if done {
		return "0s"
	}
	if !state.finished.IsZero() {
		return "--"
	}
	complete, total := pushTreeTotals(rows)
	if complete == total {
		return "waiting"
	}
	if rate <= 0 || math.IsNaN(rate) || math.IsInf(rate, 0) || now.Sub(state.lastChange) >= 30*time.Second {
		return "--"
	}
	seconds := math.Ceil(float64(total-complete) / rate)
	if seconds >= float64(int64(^uint64(0)>>1)/int64(time.Second)) {
		return "--"
	}
	return (time.Duration(seconds) * time.Second).String()
}

func clipPushLine(line string, width int) string {
	if width < 1 {
		return ""
	}
	if len(line) <= width {
		return line
	}
	if width < 4 {
		return line[:width]
	}
	return line[:width-3] + "..."
}
func renderPushFrame(label string, state pushProgress, rows []blobRow, rate float64, width, height int) string {
	// Leave one column to avoid automatic wrapping, and one row for the cursor.
	if width < 2 {
		width = 80
	}
	width--
	if height < 4 {
		height = 4
	}
	capacity := height - 4
	visible, hidden := visiblePushTree(rows, capacity)
	end := state.finished
	if end.IsZero() {
		end = time.Now()
	}
	complete, total := pushTreeTotals(rows)
	lines := []string{label, fmt.Sprintf("Total %s/%s | ETA %s | %s/s | elapsed %s | idle %s", formatPushBytes(float64(complete)), formatPushBytes(float64(total)), estimatePushRemaining(rows, state, rate, end), formatPushBytes(rate), end.Sub(state.started).Round(time.Second), end.Sub(state.lastChange).Round(time.Second))}
	nameWidth := 12
	for _, row := range rows {
		nameWidth = max(nameWidth, len(pushTreeName(row)))
	}
	now := time.Now()
	statusWidth := 0
	for _, row := range rows {
		statusWidth = max(statusWidth, len(pushRowStatus(row, now)))
	}
	compact := width < 79 || width-38-statusWidth < 12
	tiny := compact && width-19-statusWidth < 12
	if tiny {
		nameWidth = min(nameWidth, max(1, width-1-statusWidth))
	} else if compact {
		nameWidth = min(nameWidth, max(1, width-19-statusWidth))
	} else {
		nameWidth = min(nameWidth, max(1, width-38-statusWidth))
	}
	barWidth := min(14, max(4, width-nameWidth-34-statusWidth))
	for _, row := range visible {
		filled := 0
		if row.size > 0 {
			filled = int(float64(row.read) / float64(row.size) * float64(barWidth))
		}
		if row.size == 0 && completedBlob(row.phase) {
			filled = barWidth
		}
		if filled > barWidth {
			filled = barWidth
		}
		if filled < 0 {
			filled = 0
		}
		bar := strings.Repeat("=", filled) + strings.Repeat("-", barWidth-filled)
		phase := pushRowStatus(row, now)
		blobRate := float64(0)
		if !row.started.IsZero() {
			end := now
			if !row.finished.IsZero() {
				end = row.finished
			}
			if seconds := end.Sub(row.started).Seconds(); seconds > 0 {
				blobRate = float64(row.read) / seconds
			}
		}
		name := clipPushLine(pushTreeName(row), nameWidth)
		rateText := formatPushBytes(blobRate) + "/s"
		var line string
		if tiny {
			line = fmt.Sprintf("%-*s %s", nameWidth, name, phase)
		} else if compact {
			line = fmt.Sprintf("%-*s %8s/%-8s %s", nameWidth, name, formatPushBytes(float64(row.read)), formatPushBytes(float64(row.size)), phase)
		} else {
			line = fmt.Sprintf("%-*s [%s] %8s/%-8s %11s %s", nameWidth, name, bar, formatPushBytes(float64(row.read)), formatPushBytes(float64(row.size)), rateText, phase)
		}
		lines = append(lines, line)
	}
	var notes []string
	if hidden > 0 {
		notes = append(notes, fmt.Sprintf("%d more tree rows; enlarge terminal", hidden))
	}
	for _, row := range rows {
		if row.shared {
			notes = append(notes, "* shared digest; total counts each artifact once")
			break
		}
	}
	if len(notes) > 0 {
		lines = append(lines, strings.Join(notes, "; "))
	}
	for i := range lines {
		lines[i] = clipPushLine(lines[i], width)
	}
	return strings.Join(lines, "\n") + "\n"
}

// Reserve the complete status before allocating space for the bar and tree name.
func pushRowStatus(row blobRow, now time.Time) string {
	phase := row.phase
	if phase == "sending" {
		phase = fmt.Sprintf("send #%d", row.attempt)
		if !row.lastRead.IsZero() {
			if idle := now.Sub(row.lastRead); idle >= 5*time.Second {
				phase += " idle " + idle.Round(time.Second).String()
			}
		}
	}
	if phase == "waiting for registry" {
		phase = "waiting"
	}
	return phase
}
