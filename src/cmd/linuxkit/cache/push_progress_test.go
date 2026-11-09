package cache

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	namepkg "github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// This server exercises GGCR's existing uploader, not a replacement uploader.
type progressRegistry struct {
	mu          sync.Mutex
	next        int
	blobs       map[string][]byte
	sessions    map[string][]byte
	manifests   map[string][]byte
	failPatch   bool
	blockPatch  bool
	blockLookup bool
	blocked     chan struct{}
	release     chan struct{}
	once        sync.Once
}

func newProgressRegistry(t *testing.T) (*progressRegistry, *httptest.Server) {
	t.Helper()
	r := &progressRegistry{blobs: map[string][]byte{}, sessions: map[string][]byte{}, manifests: map[string][]byte{}, blocked: make(chan struct{}), release: make(chan struct{})}
	s := httptest.NewServer(http.HandlerFunc(r.serve))
	t.Cleanup(func() { close(r.release); s.Close() })
	return r, s
}
func (r *progressRegistry) serve(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path == "/v2/" {
		w.WriteHeader(200)
		return
	}
	if strings.Contains(req.URL.Path, "/manifests/") {
		if r.blockLookup {
			r.once.Do(func() { close(r.blocked) })
			select {
			case <-r.release:
			case <-req.Context().Done():
			}
			return
		}
		key := req.URL.Path[strings.LastIndex(req.URL.Path, "/")+1:]
		r.mu.Lock()
		defer r.mu.Unlock()
		if req.Method == http.MethodPut {
			data, _ := io.ReadAll(req.Body)
			r.manifests[key] = data
			w.WriteHeader(201)
			return
		}
		if data, ok := r.manifests[key]; ok {
			w.Header().Set("Content-Type", "application/vnd.oci.image.index.v1+json")
			w.Header().Set("Content-Length", fmt.Sprint(len(data)))
			sum := sha256.Sum256(data)
			w.Header().Set("Docker-Content-Digest", "sha256:"+hex.EncodeToString(sum[:]))
			w.WriteHeader(200)
			if req.Method != http.MethodHead {
				_, _ = w.Write(data)
			}
		} else {
			w.WriteHeader(404)
		}
		return
	}
	if req.Method == http.MethodHead {
		key := req.URL.Path[strings.LastIndex(req.URL.Path, "/")+1:]
		r.mu.Lock()
		_, ok := r.blobs[key]
		r.mu.Unlock()
		if ok {
			w.WriteHeader(200)
		} else {
			w.WriteHeader(404)
		}
		return
	}
	if req.Method == http.MethodPost {
		r.mu.Lock()
		r.next++
		id := fmt.Sprint(r.next)
		r.sessions[id] = nil
		r.mu.Unlock()
		w.Header().Set("Location", "/v2/test/blobs/uploads/"+id+"?secret=upload-token")
		w.WriteHeader(202)
		return
	}
	id := req.URL.Path[strings.LastIndex(req.URL.Path, "/")+1:]
	if req.Method == http.MethodPatch {
		data, _ := io.ReadAll(req.Body)
		if r.blockPatch {
			r.once.Do(func() { close(r.blocked) })
			select {
			case <-r.release:
			case <-req.Context().Done():
			}
			return
		}
		if r.failPatch {
			http.Error(w, "rejected upload", 400)
			return
		}
		r.mu.Lock()
		r.sessions[id] = append(r.sessions[id], data...)
		r.mu.Unlock()
		w.Header().Set("Location", req.URL.RequestURI())
		w.WriteHeader(202)
		return
	}
	if req.Method == http.MethodPut {
		r.mu.Lock()
		defer r.mu.Unlock()
		data := r.sessions[id]
		sum := sha256.Sum256(data)
		digest := "sha256:" + hex.EncodeToString(sum[:])
		if req.URL.Query().Get("digest") != digest {
			http.Error(w, "digest mismatch", 400)
			return
		}
		r.blobs[digest] = data
		w.WriteHeader(201)
		return
	}
	http.Error(w, "unexpected request", 400)
}

func TestPushProgressGGCRUploads(t *testing.T) {
	for _, single := range []bool{false, true} {
		t.Run(fmt.Sprint(single), func(t *testing.T) {
			f := newPushFixture(t)
			if single {
				require.NoError(t, f.p.DescriptorWrite("example.test/kernel:test", f.image))
			}
			r, server := newProgressRegistry(t)
			destination := strings.TrimPrefix(server.URL, "http://") + "/test:tag"
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			require.NoError(t, f.p.PushWithContext(ctx, "example.test/kernel:test", destination, false, true))
			r.mu.Lock()
			_, ok := r.blobs[f.layer.Digest.String()]
			r.mu.Unlock()
			require.True(t, ok, "image layer was not committed")
			// The aggregate index or image now exists; a repeated push can skip it.
			r.mu.Lock()
			count := r.next
			r.mu.Unlock()
			require.NoError(t, f.p.PushWithContext(ctx, "example.test/kernel:test", destination, false, true))
			r.mu.Lock()
			after := r.next
			r.mu.Unlock()
			require.Equal(t, count, after)
		})
	}
}

func TestPushReportsBlobStartBeforeCompletion(t *testing.T) {
	f := newPushFixture(t)
	r, server := newProgressRegistry(t)
	r.blockPatch = true
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() {
		result <- f.p.PushWithContext(ctx, "example.test/kernel:test", strings.TrimPrefix(server.URL, "http://")+"/test:tag", false, true)
	}()
	select {
	case <-r.blocked:
	case <-time.After(5 * time.Second):
		t.Fatal("upload did not start")
	}
	cancel()
	select {
	case err := <-result:
		require.ErrorIs(t, err, context.Canceled)
		require.Contains(t, err.Error(), "pending artifacts:")
		require.Contains(t, err.Error(), "attempt 1")
		require.Contains(t, err.Error(), "waiting for registry")
	case <-time.After(5 * time.Second):
		t.Fatal("push did not cancel")
	}
}

func TestPushLookupTimeout(t *testing.T) {
	f := newPushFixture(t)
	r, server := newProgressRegistry(t)
	r.blockLookup = true
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := f.p.PushWithContext(ctx, "example.test/kernel:test", strings.TrimPrefix(server.URL, "http://")+"/test:tag", false, false)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Contains(t, err.Error(), "registry lookup")
	require.Contains(t, err.Error(), "timed out")
}

func TestPushHTTPFailureContext(t *testing.T) {
	f := newPushFixture(t)
	r, server := newProgressRegistry(t)
	r.failPatch = true
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := f.p.PushWithContext(ctx, "example.test/kernel:test", strings.TrimPrefix(server.URL, "http://")+"/test:tag", false, true)
	var responseError *transport.Error
	require.ErrorAs(t, err, &responseError)
	require.Equal(t, 400, responseError.StatusCode)
	require.Contains(t, err.Error(), "PATCH")
	require.Contains(t, err.Error(), "pending artifacts:")
	require.NotContains(t, err.Error(), "upload-token")
}

func TestObservedLayerStartsAndRetries(t *testing.T) {
	f := newPushFixture(t)
	image, err := f.p.cache.ImageIndex()
	require.NoError(t, err)
	// Resolve from the fixture's nested index, whose descriptor contains the image.
	root, err := image.ImageIndex(f.root.Digest)
	require.NoError(t, err)
	child, err := root.Image(f.image.Digest)
	require.NoError(t, err)
	observer := newPushObserver()
	wrapped := observedImage{Image: child, observer: observer}
	layers, err := wrapped.Layers()
	require.NoError(t, err)
	for attempt := 1; attempt <= 2; attempt++ {
		reader, err := layers[0].Compressed()
		require.NoError(t, err)
		pending := observer.pending()
		require.Contains(t, pending, f.layer.Digest.String())
		require.Contains(t, pending, fmt.Sprintf("attempt %d", attempt))
		_, err = io.Copy(io.Discard, reader)
		require.NoError(t, err)
		require.NoError(t, reader.Close())
		require.Contains(t, observer.pending(), "waiting for registry")
	}
	observer.committed(f.layer.Digest)
	require.Empty(t, observer.pending())
	config, err := wrapped.ConfigLayer()
	require.NoError(t, err)
	reader, err := config.Compressed()
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	require.Contains(t, observer.pending(), f.config.Digest.String())
	ref, err := namepkg.ParseReference("example.test/repo:tag")
	require.NoError(t, err)
	mountable := &remote.MountableLayer{Layer: layers[0], Reference: ref}
	observed := observeLayer(mountable, observer)
	retained, ok := observed.(*remote.MountableLayer)
	require.True(t, ok)
	require.Equal(t, ref, retained.Reference)
}

func TestPushProgressMonitorDrainsAndReports(t *testing.T) {
	updates := make(chan v1.Update, 128)
	finished := make(chan struct{})
	result := make(chan pushProgress, 1)
	observer := newPushObserver()
	digest := v1.Hash{Algorithm: "sha256", Hex: strings.Repeat("a", 64)}
	observer.start(digest, 100)
	messages := make(chan string, 16)
	go func() {
		result <- monitorPush(updates, finished, observer, "test upload", time.Millisecond, func(state pushProgress) {
			select {
			case messages <- plainPushStatus("test upload", state, observer):
			default:
			}
		})
	}()
	updates <- v1.Update{Complete: 100, Total: 200}
	// GGCR's aggregate count can move backwards during layer-level retries.
	updates <- v1.Update{Complete: 20, Total: 200}
	select {
	case message := <-messages:
		require.Contains(t, message, "1 pending")
		require.Contains(t, message, "MiB processed")
	case <-time.After(time.Second):
		t.Fatal("no periodic status")
	}
	for n := 0; n < 10000; n++ {
		updates <- v1.Update{Complete: int64(n), Total: 10000}
	}
	close(updates)
	state := <-result
	require.EqualValues(t, 9999, state.complete)
	// An early failure may leave the progress channel open and still buffered.
	updates = make(chan v1.Update, 1)
	finished = make(chan struct{})
	updates <- v1.Update{Complete: 42}
	close(finished)
	state = monitorPush(updates, finished, newPushObserver(), "early failure", time.Hour, func(pushProgress) {})
	require.EqualValues(t, 42, state.complete)
}

func TestPushProgressEarlyOptionFailure(t *testing.T) {
	ref, err := namepkg.ParseReference("example.test/repo:tag")
	require.NoError(t, err)
	err = pushWithProgress(context.Background(), "test upload", newPushObserver(), []remote.Option{remote.WithJobs(0)}, func(opts ...remote.Option) error { return remote.WriteIndex(ref, empty.Index, opts...) })
	require.Error(t, err)
	require.Contains(t, err.Error(), "test upload failed")
}

func TestPushErrorRedactionAndIdentity(t *testing.T) {
	cause := &url.Error{Op: "Patch", URL: "https://user:password@example.test/uploads/123?secret=token", Err: context.DeadlineExceeded}
	now := time.Now()
	err := &pushOperationError{label: "index upload", cause: errors.Join(context.DeadlineExceeded, cause), progress: pushProgress{started: now, lastChange: now}}
	require.ErrorIs(t, err, context.DeadlineExceeded)
	var retained *url.Error
	require.ErrorAs(t, err, &retained)
	require.Same(t, cause, retained)
	require.Contains(t, err.Error(), "timed out")
	for _, secret := range []string{"password", "user:", "token"} {
		require.NotContains(t, err.Error(), secret)
	}
	require.Contains(t, cause.URL, "token", "underlying error should not be mutated")
}

func TestBlobStartLog(t *testing.T) {
	var output bytes.Buffer
	previousOutput, previousLevel := log.StandardLogger().Out, log.GetLevel()
	log.SetOutput(&output)
	log.SetLevel(log.InfoLevel)
	defer func() { log.SetOutput(previousOutput); log.SetLevel(previousLevel) }()
	observer := newPushObserver()
	digest := v1.Hash{Algorithm: "sha256", Hex: strings.Repeat("a", 64)}
	observer.start(digest, 123)
	require.Contains(t, output.String(), "starting blob upload "+digest.String())
	require.Contains(t, output.String(), "123 bytes")
}
