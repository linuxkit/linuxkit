package cache

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
)

const defaultUploadTraceInterval = 5 * time.Second
const uploadTraceWindow = 10 * time.Second

type tracePoint struct {
	at    time.Time
	bytes int64
}
type traceArtifactKey struct {
	digest  string
	attempt int
}
type traceArtifact struct {
	key                                traceArtifactKey
	kind                               string
	size, bytes, reads                 int64
	started, lastEnd, closedAt         time.Time
	readTime, gapTime, maxRead, maxGap time.Duration
	points                             []tracePoint
}

type uploadTrace struct {
	id                                               string
	started                                          time.Time
	interval                                         time.Duration
	mu                                               sync.Mutex
	artifacts                                        map[traceArtifactKey]*traceArtifact
	requests                                         map[uint64]*uploadRequestTrace
	sessions                                         map[string]string
	bindings                                         map[string]string
	connections                                      map[net.Conn]string
	nextRequest                                      uint64
	bytes                                            int64
	points                                           []tracePoint
	progressUpdates, queueHighWater, queueCapacity   int
	reportTime, maxReport                            time.Duration
	writeTime, maxWrite                              time.Duration
	writerErr                                        error
	events                                           chan map[string]interface{}
	eventMu                                          sync.Mutex
	closed                                           bool
	dropped                                          atomic.Int64
	stopSampler, samplerDone, stopWriter, writerDone chan struct{}
	closeOnce                                        sync.Once
	finalRecords                                     []map[string]interface{}
	progressChannel                                  <-chan v1.Update
}

func newUploadTrace(image, destination string, writer io.Writer, interval time.Duration) (*uploadTrace, error) {
	var id [8]byte
	if _, err := rand.Read(id[:]); err != nil {
		return nil, err
	}
	if writer == nil {
		writer = os.Stderr
	}
	if interval == 0 {
		interval = defaultUploadTraceInterval
	}
	now := time.Now()
	t := &uploadTrace{id: hex.EncodeToString(id[:]), started: now, interval: interval, artifacts: map[traceArtifactKey]*traceArtifact{}, requests: map[uint64]*uploadRequestTrace{}, sessions: map[string]string{}, bindings: map[string]string{}, connections: map[net.Conn]string{}, points: []tracePoint{{at: now}}, events: make(chan map[string]interface{}, 256), stopSampler: make(chan struct{}), samplerDone: make(chan struct{}), stopWriter: make(chan struct{}), writerDone: make(chan struct{})}
	go t.write(writer)
	t.emit(map[string]interface{}{"event": "push_start", "image": image, "destination": destination, "sample_interval_ms": milliseconds(interval), "recent_window_ms": milliseconds(uploadTraceWindow)})
	go func() {
		defer close(t.samplerDone)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				t.sample(false)
			case <-t.stopSampler:
				return
			}
		}
	}()
	return t, nil
}
func milliseconds(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }
func (t *uploadTrace) emit(record map[string]interface{}) {
	if t == nil {
		return
	}
	t.eventMu.Lock()
	defer t.eventMu.Unlock()
	if t.closed {
		return
	}
	t.stamp(record)
	select {
	case t.events <- record:
	default:
		t.dropped.Add(1)
	}
}
func (t *uploadTrace) write(writer io.Writer) {
	defer close(t.writerDone)
	encoder := json.NewEncoder(writer)
	write := func(record map[string]interface{}) {
		t.mu.Lock()
		failed := t.writerErr != nil
		t.mu.Unlock()
		if failed {
			return
		}
		start := time.Now()
		err := encoder.Encode(record)
		elapsed := time.Since(start)
		t.mu.Lock()
		t.writeTime += elapsed
		if elapsed > t.maxWrite {
			t.maxWrite = elapsed
		}
		if err != nil {
			t.writerErr = err
		}
		t.mu.Unlock()
	}
	for {
		select {
		case record := <-t.events:
			write(record)
		case <-t.stopWriter:
			for {
				select {
				case record := <-t.events:
					write(record)
				default:
					for _, record := range t.finalRecords {
						record["dropped_records"] = t.dropped.Load()
						t.mu.Lock()
						record["trace_write_ms"] = milliseconds(t.writeTime)
						record["max_trace_write_ms"] = milliseconds(t.maxWrite)
						t.mu.Unlock()
						write(record)
					}
					return
				}
			}
		}
	}
}
func (t *uploadTrace) Close(pushError error) error {
	if t == nil {
		return nil
	}
	t.closeOnce.Do(func() {
		close(t.stopSampler)
		<-t.samplerDone
		finalSample := t.sampleRecord(true)
		t.stamp(finalSample)
		outcome := "success"
		if pushError != nil {
			outcome = "error"
		}
		endRecord := map[string]interface{}{"event": "push_end", "outcome": outcome, "error_kind": traceErrorKind(pushError)}
		t.stamp(endRecord)
		t.finalRecords = []map[string]interface{}{finalSample, endRecord}
		t.eventMu.Lock()
		t.closed = true
		close(t.stopWriter)
		t.eventMu.Unlock()
		<-t.writerDone
	})
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.writerErr
}
func traceErrorKind(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.Canceled) {
		return "canceled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "deadline_exceeded"
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return "eof"
	}
	var networkError net.Error
	if errors.As(err, &networkError) {
		if networkError.Timeout() {
			return "timeout"
		}
		return "network_error"
	}
	return "error"
}
func recentTraceRate(points *[]tracePoint, now time.Time, bytes int64) (float64, float64) {
	p := append(*points, tracePoint{at: now, bytes: bytes})
	cutoff := now.Add(-uploadTraceWindow)
	for len(p) > 2 && !p[1].at.After(cutoff) {
		p = p[1:]
	}
	if len(p) > 128 {
		p = p[len(p)-128:]
	}
	*points = p
	anchor, baseline := p[0].at, float64(p[0].bytes)
	if anchor.Before(cutoff) && len(p) > 1 && p[1].at.After(cutoff) {
		fraction := float64(cutoff.Sub(anchor)) / float64(p[1].at.Sub(anchor))
		baseline += fraction * float64(p[1].bytes-p[0].bytes)
		anchor = cutoff
	}
	elapsed := now.Sub(anchor).Seconds()
	if elapsed <= 0 {
		return 0, 0
	}
	return (float64(bytes) - baseline) / elapsed, elapsed * 1000
}
func (t *uploadTrace) artifactStart(digest, kind string, size int64, attempt int) {
	if t == nil {
		return
	}
	now := time.Now()
	key := traceArtifactKey{digest, attempt}
	t.mu.Lock()
	t.artifacts[key] = &traceArtifact{key: key, kind: kind, size: size, started: now, points: []tracePoint{{at: now}}}
	t.mu.Unlock()
	t.emit(map[string]interface{}{"event": "artifact_start", "digest": digest, "kind": kind, "size": size, "attempt": attempt})
}
func (t *uploadTrace) artifactOpen(digest string, attempt int, duration time.Duration, err error) {
	if t != nil {
		t.emit(map[string]interface{}{"event": "artifact_open", "digest": digest, "attempt": attempt, "open_ms": milliseconds(duration), "error_kind": traceErrorKind(err)})
	}
}
func (t *uploadTrace) artifactRead(digest string, attempt, n int, start, end time.Time) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	a := t.artifacts[traceArtifactKey{digest, attempt}]
	if a == nil {
		return
	}
	duration := end.Sub(start)
	a.readTime += duration
	a.maxRead = max(a.maxRead, duration)
	a.reads++
	a.bytes += int64(n)
	t.bytes += int64(n)
	if !a.lastEnd.IsZero() {
		gap := start.Sub(a.lastEnd)
		if gap > 0 {
			a.gapTime += gap
			a.maxGap = max(a.maxGap, gap)
		}
	}
	a.lastEnd = end
}
func (t *uploadTrace) artifactDone(digest string, attempt int) {
	if t == nil {
		return
	}
	now := time.Now()
	t.mu.Lock()
	a := t.artifacts[traceArtifactKey{digest, attempt}]
	if a == nil || !a.closedAt.IsZero() {
		t.mu.Unlock()
		return
	}
	a.closedAt = now
	record := a.snapshot(now)
	t.mu.Unlock()
	record["event"] = "artifact_reader_done"
	t.emit(record)
}
func (a *traceArtifact) snapshot(now time.Time) map[string]interface{} {
	end := now
	if !a.closedAt.IsZero() {
		end = a.closedAt
	}
	seconds := end.Sub(a.started).Seconds()
	average := float64(0)
	if seconds > 0 {
		average = float64(a.bytes) / seconds
	}
	recent, window := recentTraceRate(&a.points, now, a.bytes)
	idle := float64(0)
	if !a.lastEnd.IsZero() {
		idle = milliseconds(now.Sub(a.lastEnd))
	}
	return map[string]interface{}{"digest": a.key.digest, "kind": a.kind, "attempt": a.key.attempt, "size": a.size, "bytes_read": a.bytes, "read_calls": a.reads, "source_read_ms": milliseconds(a.readTime), "max_source_read_ms": milliseconds(a.maxRead), "between_reads_ms": milliseconds(a.gapTime), "max_between_reads_ms": milliseconds(a.maxGap), "idle_ms": idle, "recent_bps": recent, "recent_window_ms": window, "average_bps": average, "reader_closed": !a.closedAt.IsZero()}
}
func (t *uploadTrace) progressUpdate(depth, capacity int) {
	if t == nil {
		return
	}
	t.mu.Lock()
	t.progressUpdates++
	t.queueHighWater = max(t.queueHighWater, depth)
	t.queueCapacity = capacity
	t.mu.Unlock()
}
func (t *uploadTrace) progressReport(elapsed time.Duration) {
	if t == nil {
		return
	}
	t.mu.Lock()
	t.reportTime += elapsed
	t.maxReport = max(t.maxReport, elapsed)
	t.mu.Unlock()
}
func (t *uploadTrace) sample(final bool) { t.emit(t.sampleRecord(final)) }
func (t *uploadTrace) sampleRecord(final bool) map[string]interface{} {
	now := time.Now()
	t.mu.Lock()
	keys := make([]traceArtifactKey, 0, len(t.artifacts))
	for key := range t.artifacts {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].digest == keys[j].digest {
			return keys[i].attempt < keys[j].attempt
		}
		return keys[i].digest < keys[j].digest
	})
	artifacts := make([]map[string]interface{}, 0, len(keys))
	for _, key := range keys {
		artifacts = append(artifacts, t.artifacts[key].snapshot(now))
	}
	requests := make([]*uploadRequestTrace, 0, len(t.requests))
	for _, request := range t.requests {
		requests = append(requests, request)
	}
	recent, window := recentTraceRate(&t.points, now, t.bytes)
	seconds := now.Sub(t.started).Seconds()
	average := float64(0)
	if seconds > 0 {
		average = float64(t.bytes) / seconds
	}
	queueDepth := 0
	if t.progressChannel != nil {
		queueDepth = len(t.progressChannel)
		t.queueHighWater = max(t.queueHighWater, queueDepth)
	}
	record := map[string]interface{}{"progress_queue_depth": queueDepth, "event": "sample", "final": final, "bytes_read": t.bytes, "recent_bps": recent, "recent_window_ms": window, "average_bps": average, "artifacts": artifacts, "progress_updates": t.progressUpdates, "progress_queue_high_water": t.queueHighWater, "progress_queue_capacity": t.queueCapacity, "progress_report_ms": milliseconds(t.reportTime), "max_progress_report_ms": milliseconds(t.maxReport), "trace_write_ms": milliseconds(t.writeTime), "max_trace_write_ms": milliseconds(t.maxWrite), "trace_queue_depth": len(t.events), "dropped_records": t.dropped.Load()}
	t.mu.Unlock()
	sort.Slice(requests, func(i, j int) bool { return requests[i].id < requests[j].id })
	active := make([]map[string]interface{}, 0, len(requests))
	for _, request := range requests {
		active = append(active, request.snapshot("request_active", now))
	}
	record["requests"] = active
	return record
}

type uploadTraceContextKey struct{}

func traceFromContext(ctx context.Context) *uploadTrace {
	trace, _ := ctx.Value(uploadTraceContextKey{}).(*uploadTrace)
	return trace
}
func (t *uploadTrace) operation(name string) func(error) {
	if t == nil {
		return func(error) {}
	}
	started := time.Now()
	t.emit(map[string]interface{}{"event": "operation_start", "operation": name})
	return func(err error) {
		t.emit(map[string]interface{}{"event": "operation_done", "operation": name, "duration_ms": milliseconds(time.Since(started)), "error_kind": traceErrorKind(err)})
	}
}

func (t *uploadTrace) stamp(record map[string]interface{}) {
	record["trace"] = "linuxkit-upload"
	record["schema_version"] = 1
	record["push_id"] = t.id
	record["elapsed_ms"] = milliseconds(time.Since(t.started))
}

func (t *uploadTrace) attachProgress(updates <-chan v1.Update) {
	t.mu.Lock()
	t.progressChannel = updates
	t.queueCapacity = cap(updates)
	t.mu.Unlock()
}
func (t *uploadTrace) detachProgress(updates <-chan v1.Update) {
	t.mu.Lock()
	if t.progressChannel == updates {
		t.progressChannel = nil
	}
	t.mu.Unlock()
}
