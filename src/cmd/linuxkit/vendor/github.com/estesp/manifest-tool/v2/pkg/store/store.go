package store

import (
	"bytes"
	"context"
	"io"
	"sync"
	"time"

	ccontent "github.com/containerd/containerd/content"
	"github.com/containerd/containerd/errdefs"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"oras.land/oras-go/pkg/content"
)

// ensure interface
var (
	_ ccontent.Manager  = &MemoryStore{}
	_ ccontent.Provider = &MemoryStore{}
	_ ccontent.Ingester = &MemoryStore{}
	_ ccontent.ReaderAt = sizeReaderAt{}
)

type labelStore struct {
	l      sync.RWMutex
	labels map[digest.Digest]map[string]string
}

// MemoryStore implements a simple in-memory content store for labels and
// descriptors (and associated content for manifests and configs)
type MemoryStore struct {
	store  *content.Memory
	labels labelStore
}

func newLabelStore() labelStore {
	return labelStore{
		labels: map[digest.Digest]map[string]string{},
	}
}

// NewMemoryStore creates a memory store that implements the proper
// content interfaces to support simple push/inspect operations on
// containerd's content in a memory-only context
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		store:  content.NewMemory(),
		labels: newLabelStore(),
	}
}

// Update updates mutable label field content related to a descriptor
func (m *MemoryStore) Update(ctx context.Context, info ccontent.Info, fieldpaths ...string) (ccontent.Info, error) {
	newLabels, err := m.update(info.Digest, info.Labels)
	if err != nil {
		return ccontent.Info{}, nil
	}
	info.Labels = newLabels
	return info, nil
}

// Walk is unimplemented
func (m *MemoryStore) Walk(ctx context.Context, fn ccontent.WalkFunc, filters ...string) error {
	// unimplemented
	return nil
}

func (m *MemoryStore) update(d digest.Digest, update map[string]string) (map[string]string, error) {
	m.labels.l.Lock()
	labels, ok := m.labels.labels[d]
	if !ok {
		labels = map[string]string{}
	}
	for k, v := range update {
		if v == "" {
			delete(labels, k)
		} else {
			labels[k] = v
		}
	}
	m.labels.labels[d] = labels
	m.labels.l.Unlock()

	return labels, nil
}

// Delete is unimplemented as we don't use it in the flow of manifest-tool
func (m *MemoryStore) Delete(ctx context.Context, d digest.Digest) error {
	return nil
}

// Info returns the info for a specific digest
func (m *MemoryStore) Info(ctx context.Context, d digest.Digest) (ccontent.Info, error) {
	m.labels.l.RLock()
	info := ccontent.Info{
		Digest: d,
		Labels: m.labels.labels[d],
	}
	m.labels.l.RUnlock()
	return info, nil
}

// ReaderAt returns a reader for a descriptor
func (m *MemoryStore) ReaderAt(ctx context.Context, desc ocispec.Descriptor) (ccontent.ReaderAt, error) {
	// this function is the original `ReaderAt` implementation from oras 0.9.x, copied as-is
	desc, content, ok := m.store.Get(desc)
	if !ok {
		return nil, errdefs.ErrNotFound
	}

	return sizeReaderAt{
		readAtCloser: nopCloser{
			ReaderAt: bytes.NewReader(content),
		},
		size: desc.Size,
	}, nil
}

// Writer returns a content writer given the specific options
func (m *MemoryStore) Writer(ctx context.Context, opts ...ccontent.WriterOpt) (ccontent.Writer, error) {
	// this function is the original `Writer` implementation from oras 0.9.x, copied as-is
	// given that oras-go v1.2.x has changed the signature and the implementation under a "Pusher" method
	var wOpts ccontent.WriterOpts
	for _, opt := range opts {
		if err := opt(&wOpts); err != nil {
			return nil, err
		}
	}
	desc := wOpts.Desc

	name, _ := content.ResolveName(desc)
	now := time.Now()
	return &memoryWriter{
		store:    m.store,
		buffer:   bytes.NewBuffer(nil),
		desc:     desc,
		digester: digest.Canonical.Digester(),
		status: ccontent.Status{
			Ref:       name,
			Total:     desc.Size,
			StartedAt: now,
			UpdatedAt: now,
		},
	}, nil
}

// Get returns the content for a specific descriptor
func (m *MemoryStore) Get(desc ocispec.Descriptor) (ocispec.Descriptor, []byte, bool) {
	return m.store.Get(desc)
}

// Set sets the content for a specific descriptor
func (m *MemoryStore) Set(desc ocispec.Descriptor, content []byte) {
	m.store.Set(desc, content)
}

// GetByName retrieves a descriptor based on the associated name
func (m *MemoryStore) GetByName(name string) (ocispec.Descriptor, []byte, bool) {
	return m.store.GetByName(name)
}

// Abort is not implemented or needed in this context
func (m *MemoryStore) Abort(ctx context.Context, ref string) error {
	return nil
}

// ListStatuses is not implemented or needed in this context
func (m *MemoryStore) ListStatuses(ctx context.Context, filters ...string) ([]ccontent.Status, error) {
	return []ccontent.Status{}, nil
}

// Status is not implemented or needed in this context
func (m *MemoryStore) Status(ctx context.Context, ref string) (ccontent.Status, error) {
	return ccontent.Status{}, nil
}

// the rest of this file contains the original "memoryWriter" implementation
// from oras 0.9.x to support the `Writer` function above as well as the
// `ReaderAt` implementation that uses the interfaces below

type readAtCloser interface {
	io.ReaderAt
	io.Closer
}

type sizeReaderAt struct {
	readAtCloser
	size int64
}

func (ra sizeReaderAt) Size() int64 {
	return ra.size
}

type nopCloser struct {
	io.ReaderAt
}

func (nopCloser) Close() error {
	return nil
}

type memoryWriter struct {
	store    *content.Memory
	buffer   *bytes.Buffer
	desc     ocispec.Descriptor
	digester digest.Digester
	status   ccontent.Status
}

func (w *memoryWriter) Status() (ccontent.Status, error) {
	return w.status, nil
}

// Digest returns the current digest of the content, up to the current write.
//
// Cannot be called concurrently with `Write`.
func (w *memoryWriter) Digest() digest.Digest {
	return w.digester.Digest()
}

// Write p to the transaction.
func (w *memoryWriter) Write(p []byte) (n int, err error) {
	n, err = w.buffer.Write(p)
	w.digester.Hash().Write(p[:n])
	w.status.Offset += int64(len(p))
	w.status.UpdatedAt = time.Now()
	return n, err
}

func (w *memoryWriter) Commit(ctx context.Context, size int64, expected digest.Digest, opts ...ccontent.Opt) error {
	var base ccontent.Info
	for _, opt := range opts {
		if err := opt(&base); err != nil {
			return err
		}
	}

	if w.buffer == nil {
		return errors.Wrap(errdefs.ErrFailedPrecondition, "cannot commit on closed writer")
	}
	content := w.buffer.Bytes()
	w.buffer = nil

	if size > 0 && size != int64(len(content)) {
		return errors.Wrapf(errdefs.ErrFailedPrecondition, "unexpected commit size %d, expected %d", len(content), size)
	}
	if dgst := w.digester.Digest(); expected != "" && expected != dgst {
		return errors.Wrapf(errdefs.ErrFailedPrecondition, "unexpected commit digest %s, expected %s", dgst, expected)
	}

	w.store.Set(w.desc, content)
	return nil
}

func (w *memoryWriter) Close() error {
	w.buffer = nil
	return nil
}

func (w *memoryWriter) Truncate(size int64) error {
	if size != 0 {
		return errdefs.ErrInvalidArgument
	}
	w.status.Offset = 0
	w.digester.Hash().Reset()
	w.buffer.Truncate(0)
	return nil
}
