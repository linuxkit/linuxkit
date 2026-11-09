package cache

import (
	"sort"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
)

type pushNode struct {
	kind string
	blobUpload
}

func (o *pushObserver) describeManifest(digest v1.Hash, kind string, size int64) {
	o.mu.Lock()
	defer o.mu.Unlock()
	node := o.nodes[digest]
	node.kind = kind
	node.size = size
	if node.phase == "" {
		node.phase = "queued"
	}
	o.nodes[digest] = node
}
func (o *pushObserver) manifestPhase(digest v1.Hash, phase string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if node, ok := o.nodes[digest]; ok {
		node.phase = phase
		if completedBlob(phase) {
			node.read = node.size
			node.finished = time.Now()
		}
		o.nodes[digest] = node
	}
}
func (o *pushObserver) publishedManifest(digest v1.Hash) {
	// Acceptance also confirms the referenced content. Preserve blobs already
	// marked sent while accounting for any descendants skipped by the uploader.
	o.existingManifest(digest)
	o.manifestPhase(digest, "sent")
}

func (o *pushObserver) treeSnapshot() ([]blobRow, float64) {
	o.mu.Lock()
	defer o.mu.Unlock()
	var rows []blobRow
	shown := map[v1.Hash]bool{}
	active := map[v1.Hash]bool{}
	var walk func(v1.Hash, string, string, []int)
	walk = func(digest v1.Hash, prefix, indent string, ancestors []int) {
		row := blobRow{digest: digest, prefix: prefix, ancestors: append([]int(nil), ancestors...), shared: shown[digest]}
		shown[digest] = true
		children, group := o.manifests[digest]
		if group {
			row.kind = o.nodes[digest].kind
			if row.kind == "" {
				row.kind = "manifest"
			}
			row.blobUpload = o.nodes[digest].blobUpload
		} else if state, ok := o.blobs[digest]; ok {
			row.blobUpload = state
		} else {
			row.phase = "unknown"
		}
		index := len(rows)
		rows = append(rows, row)
		if active[digest] {
			rows[index].phase = "cycle"
			for _, ancestor := range ancestors {
				rows[ancestor].phase = "cycle"
			}
			return
		}
		if !group {
			return
		}
		active[digest] = true
		defer delete(active, digest)
		path := append(append([]int(nil), ancestors...), index)
		for n, child := range children {
			branch, continuation := "|-- ", "|   "
			if n == len(children)-1 {
				branch, continuation = "`-- ", "    "
			}
			walk(child, indent+branch, indent+continuation, path)
		}
	}
	for _, root := range o.roots {
		walk(root, "", "", nil)
	}
	// Preserve visibility of any blob discovered at upload time outside the
	// initial graph, including the flat fallback used before discovery.
	var extras []v1.Hash
	for digest := range o.blobs {
		if !shown[digest] {
			extras = append(extras, digest)
		}
	}
	sort.Slice(extras, func(i, j int) bool { return extras[i].String() < extras[j].String() })
	for _, digest := range extras {
		walk(digest, "", "", nil)
	}
	seconds := time.Since(o.started).Seconds()
	rate := float64(0)
	if seconds > 0 {
		rate = float64(o.traffic) / seconds
	}
	return rows, rate
}
func pushTreeName(row blobRow) string {
	short := row.digest.Hex
	if len(short) > 12 {
		short = short[:12]
	}
	name := row.prefix + short
	if row.shared {
		name += "*"
	}
	if row.kind != "" {
		name += " " + row.kind
	}
	return name
}
func pushTreeTotals(rows []blobRow) (int64, int64) {
	seen := map[v1.Hash]bool{}
	var complete, total int64
	for _, row := range rows {
		if seen[row.digest] {
			continue
		}
		seen[row.digest] = true
		total += max(row.size, 0)
		complete += min(max(row.read, 0), max(row.size, 0))
	}
	return complete, total
}

// Select rows without reordering the tree, retaining every selected ancestor.
func visiblePushTree(rows []blobRow, capacity int) ([]blobRow, int) {
	if capacity < 0 {
		capacity = 0
	}
	if len(rows) <= capacity {
		return rows, 0
	}
	selected := make([]bool, len(rows))
	count := 0
	add := func(index int) {
		needed := 0
		if !selected[index] {
			needed++
		}
		for _, parent := range rows[index].ancestors {
			if !selected[parent] {
				needed++
			}
		}
		if count+needed > capacity {
			return
		}
		for _, parent := range rows[index].ancestors {
			if !selected[parent] {
				selected[parent] = true
				count++
			}
		}
		if !selected[index] {
			selected[index] = true
			count++
		}
	}
	for n, row := range rows {
		if row.phase != "queued" && !completedBlob(row.phase) {
			add(n)
		}
	}
	for n, row := range rows {
		if !completedBlob(row.phase) {
			add(n)
		}
	}
	for n := range rows {
		add(n)
	}
	visible := make([]blobRow, 0, count)
	for n, row := range rows {
		if selected[n] {
			visible = append(visible, row)
		}
	}
	return visible, len(rows) - len(visible)
}
