package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	v1 "github.com/google/go-containerregistry/pkg/v1"
)

// pushContentError preserves filesystem errors for errors.Is/As callers.
type pushContentError struct {
	name   string
	issues []error
	root   *pushValidationNode
}

func (e *pushContentError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "cannot push %s: local cache is incomplete or invalid (%d issue(s)); no upload attempted", e.name, len(e.issues))
	b.WriteByte('\n')
	e.root.render(&b, "", "", "")
	b.WriteString("\nComplete means local files are readable, have the expected size, and metadata is valid JSON; layer contents are not hashed.\nIncomplete marks a parent with missing or invalid descendants.\nRestore missing content or regenerate an index containing stale references before retrying.")
	return b.String()
}

func (e *pushContentError) Unwrap() []error { return e.issues }

// A node records its own validation result separately from its subtree.
// Shared subtrees are rendered once, but each descriptor remains visible.
type pushValidationNode struct {
	descriptor v1.Descriptor
	role       string
	status     string
	details    []string
	children   []*pushValidationNode
	shared     bool
	unreadable bool
}

func (n *pushValidationNode) render(b *strings.Builder, prefix, branch, continuation string) {
	d := n.descriptor
	role := n.role
	if role == "root" {
		if d.MediaType.IsIndex() {
			role = "root index"
		} else if d.MediaType.IsImage() {
			role = "root manifest"
		}
	}
	fmt.Fprintf(b, "%s%s%s %s [%s]", prefix, branch, d.Digest, role, n.status)
	if d.Platform != nil {
		fmt.Fprintf(b, " platform %s", d.Platform.String())
	}
	fmt.Fprintf(b, " (%d bytes)", d.Size)
	if target := d.Annotations["vnd.docker.reference.digest"]; target != "" {
		fmt.Fprintf(b, " attests %s", target)
	}
	if n.shared {
		b.WriteString("; shared subtree shown above")
	}
	b.WriteByte('\n')
	for _, detail := range n.details {
		fmt.Fprintf(b, "%s%s  %s\n", prefix, continuation, detail)
	}
	if n.unreadable && (d.MediaType.IsIndex() || d.MediaType.IsImage()) {
		fmt.Fprintf(b, "%s%s  descendants: NOT CHECKED (references unavailable)\n", prefix, continuation)
	}
	if n.shared {
		return
	}
	for i, child := range n.children {
		branch, next := "|-- ", "|   "
		if i == len(n.children)-1 {
			branch, next = "`-- ", "    "
		}
		child.render(b, prefix+continuation, branch, next)
	}
}

// validatePushContents checks only local content. Large layers are opened and
// checked for size, not read or hashed. It does not change the cache or use the
// network. Shared subtrees are visited once; sibling failures are accumulated.
func (p *Provider) validatePushContents(name string) error {
	_, desc, err := p.rootDescriptor(name)
	if err != nil {
		return err
	}
	var issues []error
	active := make(map[v1.Hash]bool)
	// Include type and size so conflicting descriptors are checked independently.
	type visit struct {
		digest    v1.Hash
		mediaType string
		size      int64
	}
	seen := make(map[visit]*pushValidationNode)
	var walk func(v1.Descriptor, string, string) *pushValidationNode
	walk = func(d v1.Descriptor, chain, role string) *pushValidationNode {
		node := &pushValidationNode{descriptor: d, role: role, status: "complete", unreadable: true}
		context := fmt.Sprintf("%s -> %s %s (media type %s, expected size %d", chain, role, d.Digest, d.MediaType, d.Size)
		if d.Platform != nil {
			context += ", platform " + d.Platform.String()
		}
		if target := d.Annotations["vnd.docker.reference.digest"]; target != "" {
			context += ", attests " + target
		}
		context += ")"
		fail := func(err error) {
			issues = append(issues, fmt.Errorf("%s: %w", context, err))
			node.status = "INVALID"
			if errors.Is(err, os.ErrNotExist) {
				node.status = "MISSING"
			}
			node.details = append(node.details, err.Error())
		}
		// Validate before constructing paths from cache-supplied digests.
		if _, err := v1.NewHash(d.Digest.String()); err != nil {
			fail(fmt.Errorf("invalid digest: %w", err))
			return node
		}
		if active[d.Digest] {
			fail(fmt.Errorf("descriptor cycle detected"))
			return node
		}
		key := visit{d.Digest, string(d.MediaType), d.Size}
		if prior := seen[key]; prior != nil {
			node.status, node.shared, node.unreadable = prior.status, true, false
			return node
		}
		seen[key] = node
		active[d.Digest] = true
		defer delete(active, d.Digest)
		path := filepath.Join(p.dir, "blobs", d.Digest.Algorithm, d.Digest.Hex)
		f, err := os.Open(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				fail(fmt.Errorf("missing cache blob: %w", err))
			} else {
				fail(fmt.Errorf("cannot open cache blob: %w", err))
			}
			return node
		}
		info, statErr := f.Stat()
		closeErr := f.Close()
		if statErr != nil {
			fail(fmt.Errorf("cannot stat cache blob: %w", statErr))
			return node
		}
		if closeErr != nil {
			fail(fmt.Errorf("cannot close cache blob: %w", closeErr))
			return node
		}
		if !info.Mode().IsRegular() {
			fail(fmt.Errorf("cache blob %s is not a regular file", path))
			return node
		}
		if info.Size() != d.Size {
			fail(fmt.Errorf("size mismatch at %s: found %d bytes", path, info.Size()))
			// Still inspect readable JSON to discover other missing children.
		}
		if !d.MediaType.IsIndex() && !d.MediaType.IsImage() && role != "config" {
			return node
		}
		raw, err := p.cache.Bytes(d.Digest)
		if err != nil {
			fail(fmt.Errorf("cannot read cache blob: %w", err))
			return node
		}
		switch {
		case d.MediaType.IsIndex():
			var index v1.IndexManifest
			if err := json.Unmarshal(raw, &index); err != nil {
				fail(fmt.Errorf("malformed index JSON at %s: %w", path, err))
				return node
			}
			if index.SchemaVersion != 2 {
				fail(fmt.Errorf("invalid index schema version %d at %s", index.SchemaVersion, path))
				return node
			}
			node.unreadable = false
			for i, child := range index.Manifests {
				kind := "descriptor"
				if child.MediaType.IsIndex() {
					kind = "nested index"
				} else if child.MediaType.IsImage() {
					kind = "image manifest"
				}
				node.children = append(node.children, walk(child, context, fmt.Sprintf("manifests[%d] %s", i, kind)))
			}
		case d.MediaType.IsImage():
			var manifest v1.Manifest
			if err := json.Unmarshal(raw, &manifest); err != nil {
				fail(fmt.Errorf("malformed manifest JSON at %s: %w", path, err))
				return node
			}
			if manifest.SchemaVersion != 2 {
				fail(fmt.Errorf("invalid manifest schema version %d at %s", manifest.SchemaVersion, path))
				return node
			}
			node.unreadable = false
			node.children = append(node.children, walk(manifest.Config, context, "config"))
			for i, layer := range manifest.Layers {
				node.children = append(node.children, walk(layer, context, fmt.Sprintf("layers[%d]", i)))
			}
		default:
			if !json.Valid(raw) {
				fail(fmt.Errorf("malformed config JSON at %s", path))
			}
		}
		for _, child := range node.children {
			if child.status != "complete" && node.status == "complete" {
				node.status = "incomplete"
			}
		}
		return node
	}
	root := walk(*desc, "cache reference "+name, "root")
	if len(issues) > 0 {
		return &pushContentError{name: name, issues: issues, root: root}
	}
	return nil
}
