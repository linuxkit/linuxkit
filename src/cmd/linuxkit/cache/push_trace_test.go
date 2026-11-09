package cache

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"strings"
	"sync"
	"testing"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func traceRecords(t *testing.T, data []byte) []map[string]interface{} {
	t.Helper()
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 1024), 4<<20)
	var records []map[string]interface{}
	for scanner.Scan() {
		var record map[string]interface{}
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &record))
		require.Equal(t, "linuxkit-upload", record["trace"])
		records = append(records, record)
	}
	require.NoError(t, scanner.Err())
	return records
}
func lastTraceEvent(t *testing.T, records []map[string]interface{}, event string) map[string]interface{} {
	t.Helper()
	for i := len(records) - 1; i >= 0; i-- {
		if records[i]["event"] == event {
			return records[i]
		}
	}
	t.Fatalf("missing event %s", event)
	return nil
}
func TestUploadTraceIsOptIn(t *testing.T) {
	f := newPushFixture(t)
	_, server := newProgressRegistry(t)
	var output bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, f.p.PushWithOptions(ctx, "example.test/kernel:test", strings.TrimPrefix(server.URL, "http://")+"/test:tag", false, true, PushOptions{Progress: PushProgressPlain, TraceInterval: time.Millisecond, TraceWriter: &output}))
	require.Empty(t, output.String(), "plain/non-terminal output and an interval must not enable diagnostics")
}
func TestUploadTraceRealGGCRRequestsAndSessions(t *testing.T) {
	f := newPushFixture(t)
	registry, server := newProgressRegistry(t)
	var output bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, f.p.PushWithOptions(ctx, "example.test/kernel:test", strings.TrimPrefix(server.URL, "http://")+"/test:tag", false, true, PushOptions{Progress: PushProgressPlain, TraceUpload: true, TraceWriter: &output}))
	records := traceRecords(t, output.Bytes())
	start := lastTraceEvent(t, records, "push_start")
	end := lastTraceEvent(t, records, "push_end")
	require.Equal(t, "success", end["outcome"])
	require.EqualValues(t, 0, end["dropped_records"])
	bindings := map[string]string{}
	var bodies []string
	hasTiming := false
	for _, record := range records {
		require.Equal(t, start["push_id"], record["push_id"])
		if record["event"] == "session_bind" {
			bindings[record["session_id"].(string)] = record["digest"].(string)
		}
		if record["event"] == "request_start" && record["operation"] == "upload_body" {
			require.NotEmpty(t, record["session_id"])
			bodies = append(bodies, record["session_id"].(string))
		}
		if record["event"] == "request_done" {
			_, ok := record["response_headers_ms"]
			hasTiming = hasTiming || ok
		}
	}
	require.NotEmpty(t, bodies)
	require.True(t, hasTiming)
	for _, session := range bodies {
		require.NotEmpty(t, bindings[session], "commit must link the same upload session to its digest")
	}
	registry.mu.Lock()
	var expected int
	for _, body := range registry.blobs {
		expected += len(body)
	}
	for _, body := range registry.manifests {
		expected += len(body)
	}
	registry.mu.Unlock()
	sample := lastTraceEvent(t, records, "sample")
	require.Equal(t, true, sample["final"])
	require.EqualValues(t, expected, sample["bytes_read"], "digest inspection must not count as uploaded bytes")
	require.NotEmpty(t, sample["artifacts"])
	require.Greater(t, sample["progress_updates"].(float64), float64(0))
	for _, secret := range []string{"upload-token", "?secret=", "Authorization", "test image layer"} {
		require.NotContains(t, output.String(), secret)
	}
}

type sampleTraceSink struct {
	mu      sync.Mutex
	buffer  bytes.Buffer
	samples chan map[string]interface{}
}

func (s *sampleTraceSink) Write(p []byte) (int, error) {
	s.mu.Lock()
	n, err := s.buffer.Write(p)
	s.mu.Unlock()
	var record map[string]interface{}
	if json.Unmarshal(p, &record) == nil && record["event"] == "sample" {
		select {
		case s.samples <- record:
		default:
		}
	}
	return n, err
}
func (s *sampleTraceSink) bytes() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]byte(nil), s.buffer.Bytes()...)
}
func TestUploadTraceSamplesWhileWaitingForRegistry(t *testing.T) {
	f := newPushFixture(t)
	registry, server := newProgressRegistry(t)
	registry.blockPatch = true
	sink := &sampleTraceSink{samples: make(chan map[string]interface{}, 32)}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() {
		result <- f.p.PushWithOptions(ctx, "example.test/kernel:test", strings.TrimPrefix(server.URL, "http://")+"/test:tag", false, true, PushOptions{Progress: PushProgressPlain, TraceUpload: true, TraceInterval: 20 * time.Millisecond, TraceWriter: sink})
	}()
	select {
	case <-registry.blocked:
	case <-time.After(5 * time.Second):
		t.Fatal("upload never reached PATCH")
	}
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	found := false
	for !found {
		select {
		case sample := <-sink.samples:
			for _, value := range sample["requests"].([]interface{}) {
				request := value.(map[string]interface{})
				if request["operation"] == "upload_body" && request["phase"] == "waiting_headers" {
					found = true
				}
			}
		case <-deadline.C:
			t.Fatal("no sampled request waiting for registry headers")
		}
	}
	cancel()
	select {
	case err := <-result:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(5 * time.Second):
		t.Fatal("traced push did not cancel")
	}
	end := lastTraceEvent(t, traceRecords(t, sink.bytes()), "push_end")
	require.Equal(t, "canceled", end["error_kind"])
}
func TestUploadTraceHTTPRejection(t *testing.T) {
	f := newPushFixture(t)
	registry, server := newProgressRegistry(t)
	registry.failPatch = true
	var output bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := f.p.PushWithOptions(ctx, "example.test/kernel:test", strings.TrimPrefix(server.URL, "http://")+"/test:tag", false, true, PushOptions{Progress: PushProgressPlain, TraceUpload: true, TraceWriter: &output})
	require.Error(t, err)
	records := traceRecords(t, output.Bytes())
	found := false
	for _, record := range records {
		if record["event"] == "request_done" && record["status"] == float64(400) {
			found = true
			require.Equal(t, "http_status", record["error_kind"])
		}
	}
	require.True(t, found)
	require.NotContains(t, output.String(), "upload-token")
}
func TestUploadTraceReadAndGapMetrics(t *testing.T) {
	var output bytes.Buffer
	trace, err := newUploadTrace("image", "remote", &output, time.Hour)
	require.NoError(t, err)
	digest := "sha256:" + strings.Repeat("a", 64)
	trace.artifactStart(digest, "blob", 100, 1)
	base := time.Now().Add(-time.Second)
	trace.mu.Lock()
	trace.artifacts[traceArtifactKey{digest, 1}].started = base
	trace.artifacts[traceArtifactKey{digest, 1}].points = []tracePoint{{at: base}}
	trace.mu.Unlock()
	trace.artifactRead(digest, 1, 30, base.Add(time.Millisecond), base.Add(6*time.Millisecond))
	trace.artifactRead(digest, 1, 40, base.Add(26*time.Millisecond), base.Add(29*time.Millisecond))
	trace.artifactDone(digest, 1)
	require.NoError(t, trace.Close(nil))
	record := lastTraceEvent(t, traceRecords(t, output.Bytes()), "artifact_reader_done")
	require.EqualValues(t, 70, record["bytes_read"])
	require.EqualValues(t, 2, record["read_calls"])
	require.EqualValues(t, 8, record["source_read_ms"])
	require.EqualValues(t, 20, record["between_reads_ms"])
	require.EqualValues(t, 20, record["max_between_reads_ms"])
}
func TestUploadTraceRecentRateDiffersFromAverage(t *testing.T) {
	base := time.Now()
	points := []tracePoint{{at: base, bytes: 0}}
	rate, window := recentTraceRate(&points, base.Add(5*time.Second), 1000)
	require.EqualValues(t, 200, rate)
	require.EqualValues(t, 5000, window)
	_, _ = recentTraceRate(&points, base.Add(10*time.Second), 1500)
	_, _ = recentTraceRate(&points, base.Add(15*time.Second), 2000)
	rate, window = recentTraceRate(&points, base.Add(20*time.Second), 2500)
	require.EqualValues(t, 100, rate)
	require.EqualValues(t, 10000, window)
}
func TestUploadTraceHooksPreserveContextAndRedactURLs(t *testing.T) {
	var output bytes.Buffer
	trace, err := newUploadTrace("image", "remote", &output, time.Hour)
	require.NoError(t, err)
	originalCalls := 0
	ctx := httptrace.WithClientTrace(context.Background(), &httptrace.ClientTrace{WroteRequest: func(httptrace.WroteRequestInfo) { originalCalls++ }})
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, "https://user:password@example.test/v2/repo/blobs/uploads/session-secret?token=query-secret", strings.NewReader("private body"))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer header-secret")
	request, record := trace.beginRequest(req, v1.Hash{})
	hooks := httptrace.ContextClientTrace(request.Context())
	left, right := net.Pipe()
	defer func() { require.NoError(t, left.Close()) }()
	defer func() { require.NoError(t, right.Close()) }()
	hooks.GotConn(httptrace.GotConnInfo{Conn: left, Reused: true})
	hooks.WroteHeaders()
	hooks.WroteRequest(httptrace.WroteRequestInfo{})
	response := &http.Response{StatusCode: 202, Header: http.Header{"Location": []string{"?token=new-secret"}}, Proto: "HTTP/2.0", Body: io.NopCloser(strings.NewReader("private response"))}
	record.response(request, response, nil)
	_, err = io.Copy(io.Discard, response.Body)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	require.NoError(t, trace.Close(nil))
	require.Equal(t, 1, originalCalls)
	for _, secret := range []string{"password", "user:", "query-secret", "header-secret", "session-secret", "new-secret", "private body", "private response"} {
		require.NotContains(t, output.String(), secret)
	}
	done := lastTraceEvent(t, traceRecords(t, output.Bytes()), "request_done")
	require.Equal(t, true, done["connection_reused"])
	require.Equal(t, "HTTP/2.0", done["http_protocol"])
	require.NotContains(t, done, "tls_ms")
	require.NotContains(t, done, "dns_ms")
}

func TestUploadTraceEarlyResponseHasNoInventedWait(t *testing.T) {
	var output bytes.Buffer
	trace, err := newUploadTrace("image", "remote", &output, time.Hour)
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPatch, "http://example.test/v2/repo/blobs/uploads/one", strings.NewReader("data"))
	require.NoError(t, err)
	request, record := trace.beginRequest(req, v1.Hash{})
	response := &http.Response{StatusCode: 400, Header: http.Header{}, Body: http.NoBody}
	record.response(request, response, nil)
	require.NoError(t, trace.Close(nil))
	done := lastTraceEvent(t, traceRecords(t, output.Bytes()), "request_done")
	require.Equal(t, true, done["early_response"])
	require.NotContains(t, done, "response_wait_ms")
	require.Equal(t, "http_status", done["error_kind"])
}

type blockedTraceWriter struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
	buffer  bytes.Buffer
}

func (w *blockedTraceWriter) Write(p []byte) (int, error) {
	w.once.Do(func() { close(w.started) })
	<-w.release
	return w.buffer.Write(p)
}
func TestUploadTraceQueueIsBoundedAndNonblocking(t *testing.T) {
	writer := &blockedTraceWriter{started: make(chan struct{}), release: make(chan struct{})}
	trace, err := newUploadTrace("image", "remote", writer, time.Hour)
	require.NoError(t, err)
	<-writer.started
	finished := make(chan struct{})
	go func() {
		for i := 0; i < 2000; i++ {
			trace.emit(map[string]interface{}{"event": "test"})
		}
		close(finished)
	}()
	select {
	case <-finished:
	case <-time.After(time.Second):
		close(writer.release)
		t.Fatal("trace producer blocked on output")
	}
	require.Positive(t, trace.dropped.Load())
	close(writer.release)
	require.NoError(t, trace.Close(nil))
	end := lastTraceEvent(t, traceRecords(t, writer.buffer.Bytes()), "push_end")
	require.Greater(t, end["dropped_records"].(float64), float64(0))
}

type failedTraceWriter struct{}

func (failedTraceWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
func TestUploadTraceWriterFailureIsReported(t *testing.T) {
	trace, err := newUploadTrace("image", "remote", failedTraceWriter{}, time.Hour)
	require.NoError(t, err)
	require.ErrorIs(t, trace.Close(nil), io.ErrClosedPipe)
}
func TestTraceOutputPolicy(t *testing.T) {
	logger := log.StandardLogger()
	out := logger.Out
	defer logger.SetOutput(out)
	var output bytes.Buffer
	logger.SetOutput(&output)
	require.NoError(t, (PushOptions{TraceUpload: true}).Validate())
	require.NoError(t, (PushOptions{Progress: PushProgressPlain, TraceUpload: true}).Validate())
	require.ErrorContains(t, (PushOptions{Progress: PushProgressTTY, TraceUpload: true}).Validate(), "incompatible")
	require.ErrorContains(t, (PushOptions{TraceInterval: -time.Second}).Validate(), "must not be negative")
	var nilTrace *uploadTrace
	require.NoError(t, nilTrace.Close(errors.New("unused")))
}

func TestUploadTraceSeesAFullProgressChannel(t *testing.T) {
	var output bytes.Buffer
	trace, err := newUploadTrace("image", "remote", &output, time.Hour)
	require.NoError(t, err)
	updates := make(chan v1.Update, 4)
	trace.attachProgress(updates)
	for i := 0; i < 4; i++ {
		updates <- v1.Update{Complete: int64(i)}
	}
	// Sampling reads live occupancy without depending on the consumer running.
	trace.sample(false)
	trace.detachProgress(updates)
	require.NoError(t, trace.Close(nil))
	records := traceRecords(t, output.Bytes())
	found := false
	for _, record := range records {
		if record["event"] == "sample" && record["final"] == false {
			found = true
			require.EqualValues(t, 4, record["progress_queue_depth"])
			require.EqualValues(t, 4, record["progress_queue_high_water"])
		}
	}
	require.True(t, found)
}

func TestUploadTraceQueryOnlySessionsAndChangedLocations(t *testing.T) {
	var output bytes.Buffer
	trace, err := newUploadTrace("image", "remote", &output, time.Hour)
	require.NoError(t, err)
	init, err := http.NewRequest(http.MethodPost, "http://example.test/v2/repo/blobs/uploads/", nil)
	require.NoError(t, err)
	request, first := trace.beginRequest(init, v1.Hash{})
	first.response(request, &http.Response{StatusCode: 202, Header: http.Header{"Location": []string{"/upload?state=one-secret"}}, Body: http.NoBody}, nil)
	_, second := trace.beginRequest(init, v1.Hash{})
	second.response(init, &http.Response{StatusCode: 202, Header: http.Header{"Location": []string{"/upload?state=two-secret"}}, Body: http.NoBody}, nil)
	require.NotEqual(t, first.session, second.session)
	patch, err := http.NewRequest(http.MethodPatch, "http://example.test/upload?state=one-secret", nil)
	require.NoError(t, err)
	_, body := trace.beginRequest(patch, v1.Hash{})
	require.Equal(t, first.session, body.session)
	body.response(patch, &http.Response{StatusCode: 202, Header: http.Header{"Location": []string{"/upload?state=updated-secret"}}, Body: http.NoBody}, nil)
	digest := "sha256:" + strings.Repeat("a", 64)
	commit, err := http.NewRequest(http.MethodPut, "http://example.test/upload?digest="+digest+"&state=updated-secret", nil)
	require.NoError(t, err)
	_, closing := trace.beginRequest(commit, v1.Hash{})
	require.Equal(t, first.session, closing.session)
	closing.response(commit, &http.Response{StatusCode: 201, Header: http.Header{}, Body: http.NoBody}, nil)
	require.NoError(t, trace.Close(nil))
	for _, secret := range []string{"one-secret", "two-secret", "updated-secret"} {
		require.NotContains(t, output.String(), secret)
	}
}
