package cache

import (
	"strconv"
	"strings"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/util"
	log "github.com/sirupsen/logrus"
)

func completedBlob(phase string) bool {
	return phase == "sent" || phase == "exists" || phase == "mounted" || phase == "skipped"
}
func (o *pushObserver) rememberManifest(digest v1.Hash, children []v1.Hash) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.manifests[digest] = append([]v1.Hash(nil), children...)
	if _, ok := o.nodes[digest]; !ok {
		o.nodes[digest] = pushNode{kind: "manifest", blobUpload: blobUpload{phase: "queued"}}
	}
}
func (o *pushObserver) completeBlobLocked(digest v1.Hash, phase string) {
	state, ok := o.blobs[digest]
	if !ok {
		return
	}
	// A second manifest existence check must not relabel a blob we just sent.
	if !completedBlob(state.phase) {
		state.phase = phase
		state.read = state.size
		state.finished = time.Now()
		o.blobs[digest] = state
	}
	delete(o.active, digest)
}
func (o *pushObserver) mounted(digest v1.Hash) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.completeBlobLocked(digest, "mounted")
}

// coveredLocked requires every reachable object to have a table row. When complete
// is true, every object must also be represented by a completed state. Unknown
// children and cycles keep the upstream event visible rather than hiding it.
func (o *pushObserver) coveredLocked(digest v1.Hash, complete bool) bool {
	active := map[v1.Hash]bool{}
	var walk func(v1.Hash) bool
	walk = func(d v1.Hash) bool {
		if active[d] {
			return false
		}
		if children, ok := o.manifests[d]; ok {
			node, represented := o.nodes[d]
			if !represented || (complete && !completedBlob(node.phase)) {
				return false
			}
			active[d] = true
			defer delete(active, d)
			for _, child := range children {
				if !walk(child) {
					return false
				}
			}
			return true
		}
		state, ok := o.blobs[d]
		return ok && (!complete || completedBlob(state.phase))
	}
	return walk(digest)
}
func (o *pushObserver) existingManifest(digest v1.Hash) bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	if _, known := o.manifests[digest]; !known || !o.coveredLocked(digest, false) {
		return false
	}
	seen := map[v1.Hash]bool{}
	var mark func(v1.Hash)
	mark = func(d v1.Hash) {
		if seen[d] {
			return
		}
		seen[d] = true
		if children, ok := o.manifests[d]; ok {
			node := o.nodes[d]
			if node.phase != "sent" {
				node.phase = "exists"
				node.read = node.size
				node.finished = time.Now()
				o.nodes[d] = node
			}
			for _, child := range children {
				mark(child)
			}
			return
		}
		o.completeBlobLocked(d, "exists")
	}
	mark(digest)
	return true
}

// Parse only known GGCR completion forms. Warnings, unknown messages and events
// for content not completely represented by this display remain visible.
func (o *pushObserver) redundantCompletion(message string) bool {
	var value string
	manifest := false
	for _, prefix := range []string{"existing blob: ", "pushed blob: ", "mounted blob: ", "existing manifest: "} {
		if strings.HasPrefix(message, prefix) {
			value = strings.TrimPrefix(message, prefix)
			manifest = prefix == "existing manifest: "
			break
		}
	}
	if value == "" {
		fields := strings.Fields(message)
		if len(fields) != 5 || !strings.HasSuffix(fields[0], ":") || fields[1] != "digest:" || fields[3] != "size:" {
			return false
		}
		if size, err := strconv.ParseInt(fields[4], 10, 64); err != nil || size < 0 {
			return false
		}
		value = fields[2]
		manifest = true
	}
	if manifest {
		if i := strings.LastIndex(value, "@"); i >= 0 {
			value = value[i+1:]
		}
	}
	digest, err := v1.NewHash(value)
	if err != nil {
		return false
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if manifest {
		if _, ok := o.manifests[digest]; !ok {
			return false
		}
	} else {
		if _, ok := o.blobs[digest]; !ok {
			return false
		}
	}
	return o.coveredLocked(digest, true)
}

type pushCompletionFormatter struct {
	log.Formatter
	observer *pushObserver
}

func (f *pushCompletionFormatter) Format(entry *log.Entry) ([]byte, error) {
	if entry.Level == log.InfoLevel && entry.Data[util.GGCRProgressField] == true && f.observer.redundantCompletion(entry.Message) {
		return nil, nil
	}
	return f.Formatter.Format(entry)
}
