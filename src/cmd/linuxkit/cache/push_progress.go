package cache

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	log "github.com/sirupsen/logrus"
)

const (
	// DefaultPushTimeout bounds the complete push, not each individual request.
	DefaultPushTimeout   = 30 * time.Minute
	pushProgressInterval = 30 * time.Second
)

type blobUpload struct {
	size, read                  int64
	attempt                     int
	phase                       string
	started, lastRead, finished time.Time
}

// pushObserver reports local reads separately from registry acknowledgement.
// GGCR's progress channel supplies aggregate counters, including skipped blobs.
type pushObserver struct {
	mu           sync.Mutex
	active       map[v1.Hash]blobUpload
	attempts     map[v1.Hash]int
	blobs        map[v1.Hash]blobUpload
	manifests    map[v1.Hash][]v1.Hash
	nodes        map[v1.Hash]pushNode
	roots        []v1.Hash
	interactive  bool
	traffic      int64
	started      time.Time
	lastRead     time.Time
	progressMode PushProgressMode
	trace        *uploadTrace
}

func newPushObserver() *pushObserver {
	return &pushObserver{active: make(map[v1.Hash]blobUpload), attempts: make(map[v1.Hash]int), blobs: make(map[v1.Hash]blobUpload), manifests: make(map[v1.Hash][]v1.Hash), nodes: make(map[v1.Hash]pushNode), started: time.Now()}
}
func (o *pushObserver) start(digest v1.Hash, size int64) int {
	o.mu.Lock()
	o.attempts[digest]++
	attempt := o.attempts[digest]
	now := time.Now()
	o.active[digest] = blobUpload{size: size, attempt: attempt, phase: "sending", started: now, lastRead: now}
	o.blobs[digest] = o.active[digest]
	interactive := o.interactive
	o.mu.Unlock()
	if o.trace != nil {
		o.trace.artifactStart(digest.String(), "blob", size, attempt)
	}
	if !interactive {
		log.Infof("starting blob upload %s (%d bytes), attempt %d", digest, size, attempt)
	}
	return attempt
}
func (o *pushObserver) read(digest v1.Hash, attempt int, n int, closing bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	state, ok := o.active[digest]
	if !ok || state.attempt != attempt {
		return
	}
	state.read += int64(n)
	o.traffic += int64(n)
	if n > 0 {
		state.lastRead = time.Now()
	}
	if closing {
		state.phase = "waiting for registry"
	}
	o.active[digest] = state
	o.blobs[digest] = state
	if n > 0 {
		o.lastRead = state.lastRead
	}
}
func (o *pushObserver) phase(digest v1.Hash, phase string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if state, ok := o.active[digest]; ok {
		state.phase = phase
		o.active[digest] = state
		o.blobs[digest] = state
	} else if state, ok := o.blobs[digest]; ok {
		state.phase = phase
		o.blobs[digest] = state
	}
}

func (o *pushObserver) committed(digest v1.Hash) {
	o.mu.Lock()
	defer o.mu.Unlock()
	phase := "exists"
	if state, ok := o.blobs[digest]; ok && state.attempt > 0 {
		phase = "sent"
	}
	o.completeBlobLocked(digest, phase)
}
func (o *pushObserver) pending() string {
	o.mu.Lock()
	defer o.mu.Unlock()
	var entries []string
	for digest, state := range o.active {
		entries = append(entries, fmt.Sprintf("%s: %d/%d bytes read, attempt %d, %s", digest, state.read, state.size, state.attempt, state.phase))
	}
	for digest, node := range o.nodes {
		if node.attempt > 0 && !completedBlob(node.phase) {
			entries = append(entries, fmt.Sprintf("%s (%s): %d/%d bytes read, attempt %d, %s", digest, node.kind, node.read, node.size, node.attempt, node.phase))
		}
	}
	sort.Strings(entries)
	return strings.Join(entries, "; ")
}

// Observe commit responses without changing GGCR's requests, auth or retries.
type pushTransport struct {
	http.RoundTripper
	observer *pushObserver
}

func (t pushTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var digest v1.Hash
	var track bool
	var publishing v1.Hash
	if req.Method == http.MethodPut && strings.Contains(req.URL.Path, "/manifests/") {
		publishing, _ = v1.NewHash(req.URL.Path[strings.LastIndex(req.URL.Path, "/")+1:])
		if req.GetBody != nil {
			if body, err := req.GetBody(); err == nil {
				if hash, _, err := v1.SHA256(body); err == nil {
					publishing = hash
				}
				_ = body.Close()
			}
		}
		if publishing.Hex != "" {
			request := req.Clone(req.Context())
			if req.Body != nil {
				request.Body = t.observer.manifestBody(publishing, req.ContentLength, req.Body)
			}
			if req.GetBody != nil {
				original := req.GetBody
				request.GetBody = func() (io.ReadCloser, error) {
					body, err := original()
					if err != nil {
						return nil, err
					}
					return t.observer.manifestBody(publishing, request.ContentLength, body), nil
				}
			}
			req = request
		}
	}
	if req.Method == http.MethodPut {
		digest, _ = v1.NewHash(req.URL.Query().Get("digest"))
		track = digest.Hex != "" && strings.Contains(req.URL.Path, "/blobs/uploads/")
		if track {
			t.observer.phase(digest, "committing")
			log.Debugf("committing blob %s", digest)
		}
	} else if req.Method == http.MethodHead && strings.Contains(req.URL.Path, "/blobs/") {
		digest, _ = v1.NewHash(req.URL.Path[strings.LastIndex(req.URL.Path, "/")+1:])
		track = digest.Hex != ""
		if track {
			t.observer.phase(digest, "checking")
			log.Debugf("checking remote blob %s", digest)
		}
	}
	var requestTrace *uploadRequestTrace
	if t.observer.trace != nil {
		req, requestTrace = t.observer.trace.beginRequest(req, publishing)
	}
	resp, err := t.RoundTripper.RoundTrip(req)
	if requestTrace != nil {
		requestTrace.response(req, resp, err)
	}
	if err != nil {
		return resp, err
	}
	if publishing.Hex != "" && (resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusAccepted) {
		t.observer.publishedManifest(publishing)
	}
	if track && (resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK) {
		t.observer.committed(digest)
	}
	if req.Method == http.MethodHead && strings.Contains(req.URL.Path, "/manifests/") && resp.StatusCode == http.StatusOK && resp.ContentLength >= 0 && resp.Header.Get("Content-Type") != "" {
		found, parseErr := v1.NewHash(resp.Header.Get("Docker-Content-Digest"))
		requested, digestErr := v1.NewHash(req.URL.Path[strings.LastIndex(req.URL.Path, "/")+1:])
		if parseErr == nil && (digestErr != nil || found == requested) {
			t.observer.existingManifest(found)
		}
	}
	if req.Method == http.MethodPost && strings.Contains(req.URL.Path, "/blobs/uploads/") && resp.StatusCode == http.StatusCreated {
		if mounted, parseErr := v1.NewHash(req.URL.Query().Get("mount")); parseErr == nil {
			t.observer.mounted(mounted)
		}
	}
	return resp, err
}

type observedLayer struct {
	v1.Layer
	observer *pushObserver
}

func (l observedLayer) Compressed() (io.ReadCloser, error) {
	digest, err := l.Digest()
	if err != nil {
		return nil, err
	}
	size, err := l.Size()
	if err != nil {
		return nil, err
	}
	attempt := l.observer.start(digest, size)
	var opening time.Time
	if l.observer.trace != nil {
		opening = time.Now()
	}
	reader, err := l.Layer.Compressed()
	if l.observer.trace != nil {
		l.observer.trace.artifactOpen(digest.String(), attempt, time.Since(opening), err)
	}
	if err != nil {
		return nil, fmt.Errorf("opening blob %s for upload (attempt %d): %w", digest, attempt, err)
	}
	return &observedBlobReader{ReadCloser: reader, observer: l.observer, digest: digest, attempt: attempt}, nil
}
func (l observedLayer) Descriptor() (*v1.Descriptor, error) { return partial.Descriptor(l.Layer) }
func (l observedLayer) Exists() (bool, error)               { return partial.Exists(l.Layer) }

type observedBlobReader struct {
	io.ReadCloser
	observer *pushObserver
	digest   v1.Hash
	attempt  int
}

func (r *observedBlobReader) Read(p []byte) (int, error) {
	var start time.Time
	if r.observer.trace != nil {
		start = time.Now()
	}
	n, err := r.ReadCloser.Read(p)
	if r.observer.trace != nil {
		r.observer.trace.artifactRead(r.digest.String(), r.attempt, n, start, time.Now())
	}
	r.observer.read(r.digest, r.attempt, n, err == io.EOF)
	return n, err
}
func (r *observedBlobReader) Close() error {
	r.observer.read(r.digest, r.attempt, 0, true)
	err := r.ReadCloser.Close()
	if r.observer.trace != nil {
		r.observer.trace.artifactDone(r.digest.String(), r.attempt)
	}
	return err
}
func observeLayer(layer v1.Layer, observer *pushObserver) v1.Layer {
	// GGCR checks this concrete type to retain cross-repository mounting.
	if mountable, ok := layer.(*remote.MountableLayer); ok {
		return &remote.MountableLayer{Layer: observedLayer{Layer: mountable.Layer, observer: observer}, Reference: mountable.Reference}
	}
	return observedLayer{Layer: layer, observer: observer}
}

type observedImage struct {
	v1.Image
	observer *pushObserver
}

func (i observedImage) Layers() ([]v1.Layer, error) {
	layers, err := i.Image.Layers()
	if err != nil {
		return nil, err
	}
	result := make([]v1.Layer, len(layers))
	for n, l := range layers {
		result[n] = observeLayer(l, i.observer)
	}
	return result, nil
}
func (i observedImage) LayerByDigest(h v1.Hash) (v1.Layer, error) {
	layer, err := i.Image.LayerByDigest(h)
	if err != nil {
		return nil, err
	}
	return observeLayer(layer, i.observer), nil
}
func (i observedImage) LayerByDiffID(h v1.Hash) (v1.Layer, error) {
	layer, err := i.Image.LayerByDiffID(h)
	if err != nil {
		return nil, err
	}
	return observeLayer(layer, i.observer), nil
}
func (i observedImage) ConfigLayer() (v1.Layer, error) {
	layer, err := partial.ConfigLayer(i.Image)
	if err != nil {
		return nil, err
	}
	return observeLayer(layer, i.observer), nil
}
func (i observedImage) Descriptor() (*v1.Descriptor, error) { return partial.Descriptor(i.Image) }

type pushImageIndex interface{ v1.ImageIndex }

type observedIndex struct {
	pushImageIndex
	observer *pushObserver
}

func (i observedIndex) Image(h v1.Hash) (v1.Image, error) {
	image, err := i.pushImageIndex.Image(h)
	if err != nil {
		return nil, err
	}
	return observedImage{Image: image, observer: i.observer}, nil
}
func (i observedIndex) ImageIndex(h v1.Hash) (v1.ImageIndex, error) {
	index, err := i.pushImageIndex.ImageIndex(h)
	if err != nil {
		return nil, err
	}
	return observedIndex{pushImageIndex: index, observer: i.observer}, nil
}
func (i observedIndex) Manifests() ([]partial.Describable, error) {
	children, err := partial.Manifests(i.pushImageIndex)
	if err != nil {
		return nil, err
	}
	result := make([]partial.Describable, len(children))
	for n, child := range children {
		switch value := child.(type) {
		case v1.Image:
			result[n] = observedImage{Image: value, observer: i.observer}
		case v1.ImageIndex:
			result[n] = observedIndex{pushImageIndex: value, observer: i.observer}
		case v1.Layer:
			result[n] = observeLayer(value, i.observer)
		default:
			result[n] = child
		}
	}
	return result, nil
}
func (i observedIndex) Descriptor() (*v1.Descriptor, error) {
	return partial.Descriptor(i.pushImageIndex)
}

type pushProgress struct {
	complete, total     int64
	started, lastChange time.Time
	finished            time.Time
	err                 error
}

func (p *pushProgress) update(u v1.Update) {
	if u.Error != nil {
		p.err = u.Error
		return
	}
	if u.Complete != p.complete {
		p.lastChange = time.Now()
	}
	p.complete, p.total = u.Complete, u.Total
}
func (p pushProgress) summary() string {
	now := p.finished
	if now.IsZero() {
		now = time.Now()
	}
	timing := fmt.Sprintf("elapsed %s; last byte progress %s ago", now.Sub(p.started).Round(time.Second), now.Sub(p.lastChange).Round(time.Second))
	if p.total == 0 && p.complete == 0 {
		return "no transfer bytes reported; " + timing
	}
	return fmt.Sprintf("%.1f/%.1f MiB processed; %s", float64(p.complete)/(1<<20), float64(p.total)/(1<<20), timing)
}

func monitorPush(updates <-chan v1.Update, finished <-chan struct{}, observer *pushObserver, label string, interval time.Duration, report func(pushProgress)) (state pushProgress) {
	now := time.Now()
	if observer.trace != nil {
		observer.trace.attachProgress(updates)
		defer observer.trace.detachProgress(updates)
	}
	state = pushProgress{started: now, lastChange: now}
	defer func() { state.finished = time.Now() }()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case update, open := <-updates:
			if !open {
				return state
			}
			state.update(update)
			if observer.trace != nil {
				observer.trace.progressUpdate(len(updates), cap(updates))
			}
		case <-ticker.C:
			if observer.trace == nil {
				report(state)
			} else {
				start := time.Now()
				report(state)
				observer.trace.progressReport(time.Since(start))
			}
		case <-finished:
			// Early GGCR option errors may return without closing the channel.
			// Drain queued updates without requiring the library to close it.
			for {
				select {
				case u, open := <-updates:
					if !open {
						return state
					}
					state.update(u)
					if observer.trace != nil {
						observer.trace.progressUpdate(len(updates), cap(updates))
					}
				default:
					return state
				}
			}
		}
	}
}

func pushWithProgress(ctx context.Context, label string, observer *pushObserver, options []remote.Option, write func(...remote.Option) error) (resultError error) {
	done := observer.trace.operation(label)
	defer func() { done(resultError) }()
	updates := make(chan v1.Update, 128)
	finished := make(chan struct{})
	result := make(chan pushProgress, 1)
	display := newPushDisplay(observer, label)
	defer display.Close()
	log.Debugf("starting %s", label)
	interval := pushProgressInterval
	if display.interactive() {
		interval = 200 * time.Millisecond
	}
	go func() { result <- monitorPush(updates, finished, observer, label, interval, display.Report) }()
	// A separate WithProgress channel is required for each operation: GGCR owns
	// and closes it. Do not reuse its option for lookups or architecture tags.
	opts := append(append([]remote.Option(nil), options...), remote.WithProgress(updates))
	err := write(opts...)
	close(finished)
	state := <-result
	state = observer.progressState(state)
	if err == nil {
		err = state.err
	}
	if err != nil {
		if ctx.Err() != nil {
			err = errors.Join(ctx.Err(), err)
		}
		display.Finish(state, false)
		return &pushOperationError{label: label, progress: state, pending: observer.pending(), cause: err}
	}
	display.Finish(state, true)
	log.Infof("finished %s: %s", label, observer.progressState(state).summary())
	return nil
}

// Keep the original cause available while redacting network URL query values.
type pushOperationError struct {
	label    string
	progress pushProgress
	pending  string
	cause    error
}

func (e *pushOperationError) Unwrap() error { return e.cause }
func (e *pushOperationError) Error() string {
	status := "failed"
	if errors.Is(e.cause, context.DeadlineExceeded) {
		status = "timed out"
	} else if errors.Is(e.cause, context.Canceled) {
		status = "canceled"
	}
	message := fmt.Sprintf("%s %s (%s)", e.label, status, e.progress.summary())
	if e.pending != "" {
		message += "; pending artifacts: " + e.pending
	}
	return message + ": " + pushErrorText(e.cause)
}
func pushErrorText(err error) string {
	// Join errors can include both the context failure and the HTTP failure.
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		var messages []string
		for _, child := range joined.Unwrap() {
			messages = append(messages, pushErrorText(child))
		}
		return strings.Join(messages, "; ")
	}
	var requestError *url.Error
	if errors.As(err, &requestError) {
		copy := *requestError
		if u, parseErr := url.Parse(copy.URL); parseErr == nil {
			if u.RawQuery != "" {
				u.RawQuery = "REDACTED"
			}
			u.User = nil
			copy.URL = u.String()
		}
		return copy.Error()
	}
	return err.Error()
}
