package cache

import (
	"io"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	log "github.com/sirupsen/logrus"
)

// Metadata objects have the same local-read progress as blobs. Request-body
// observation is scoped to the known manifest digest and leaves GGCR's upload,
// redirect and retry behavior intact.
func (o *pushObserver) manifestBody(digest v1.Hash, size int64, body io.ReadCloser) io.ReadCloser {
	o.mu.Lock()
	node, known := o.nodes[digest]
	if !known {
		o.mu.Unlock()
		return body
	}
	node.attempt++
	attempt := node.attempt
	if size >= 0 {
		node.size = size
	}
	now := time.Now()
	node.started = now
	node.lastRead = now
	node.finished = time.Time{}
	node.read = 0
	node.phase = "sending"
	o.nodes[digest] = node
	interactive := o.interactive
	o.mu.Unlock()
	if o.trace != nil {
		o.trace.artifactStart(digest.String(), node.kind, node.size, attempt)
	}
	if !interactive {
		log.Infof("starting %s upload %s (%d bytes), attempt %d", node.kind, digest, node.size, attempt)
	}
	return &manifestProgressReader{ReadCloser: body, observer: o, digest: digest, attempt: attempt}
}
func (o *pushObserver) readManifest(digest v1.Hash, attempt, n int, closing bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	node, known := o.nodes[digest]
	if !known || node.attempt != attempt || completedBlob(node.phase) {
		return
	}
	node.read += int64(n)
	o.traffic += int64(n)
	if n > 0 {
		node.lastRead = time.Now()
		o.lastRead = node.lastRead
	}
	if closing {
		node.phase = "waiting for registry"
	}
	o.nodes[digest] = node
}

type manifestProgressReader struct {
	io.ReadCloser
	observer *pushObserver
	digest   v1.Hash
	attempt  int
}

func (r *manifestProgressReader) Read(p []byte) (int, error) {
	var start time.Time
	if r.observer.trace != nil {
		start = time.Now()
	}
	n, err := r.ReadCloser.Read(p)
	if r.observer.trace != nil {
		r.observer.trace.artifactRead(r.digest.String(), r.attempt, n, start, time.Now())
	}
	r.observer.readManifest(r.digest, r.attempt, n, err == io.EOF)
	return n, err
}
func (r *manifestProgressReader) Close() error {
	r.observer.readManifest(r.digest, r.attempt, 0, true)
	err := r.ReadCloser.Close()
	if r.observer.trace != nil {
		r.observer.trace.artifactDone(r.digest.String(), r.attempt)
	}
	return err
}
