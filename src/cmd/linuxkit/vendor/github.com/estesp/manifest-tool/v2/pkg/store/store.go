package store

import (
	"context"
	"sync"

	ccontent "github.com/containerd/containerd/content"
	"github.com/deislabs/oras/pkg/content"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// ensure interface
var (
	_ ccontent.Manager  = &MemoryStore{}
	_ ccontent.Provider = &MemoryStore{}
	_ ccontent.Ingester = &MemoryStore{}
)

type labelStore struct {
	l      sync.Mutex
	labels map[digest.Digest]map[string]string
}

// MemoryStore implements a simple in-memory content store for labels and
// descriptors (and associated content for manifests and configs)
type MemoryStore struct {
	store  *content.Memorystore
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
		store:  content.NewMemoryStore(),
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
	info := ccontent.Info{
		Digest: d,
		Labels: m.labels.labels[d],
	}
	return info, nil
}

// ReaderAt returns a reader for a descriptor
func (m *MemoryStore) ReaderAt(ctx context.Context, desc ocispec.Descriptor) (ccontent.ReaderAt, error) {
	return m.store.ReaderAt(ctx, desc)
}

// Writer returns a content writer given the specific options
func (m *MemoryStore) Writer(ctx context.Context, opts ...ccontent.WriterOpt) (ccontent.Writer, error) {
	return m.store.Writer(ctx, opts...)
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
