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
}

func (e *pushContentError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "cannot push %s: local cache is incomplete or invalid (%d issue(s)); no upload attempted", e.name, len(e.issues))
	for _, issue := range e.issues {
		fmt.Fprintf(&b, "\n- %s", issue)
	}
	b.WriteString("\nDescendants of unreadable indexes or manifests could not be checked. Restore the referenced content or rebuild and regenerate the affected index before retrying.")
	return b.String()
}

func (e *pushContentError) Unwrap() []error { return e.issues }

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
	seen := make(map[visit]bool)
	var walk func(v1.Descriptor, string, string)
	walk = func(d v1.Descriptor, chain, role string) {
		context := fmt.Sprintf("%s -> %s %s (media type %s, expected size %d", chain, role, d.Digest, d.MediaType, d.Size)
		if d.Platform != nil {
			context += ", platform " + d.Platform.String()
		}
		if target := d.Annotations["vnd.docker.reference.digest"]; target != "" {
			context += ", attests " + target
		}
		context += ")"
		fail := func(err error) { issues = append(issues, fmt.Errorf("%s: %w", context, err)) }
		// Validate before constructing paths from cache-supplied digests.
		if _, err := v1.NewHash(d.Digest.String()); err != nil {
			fail(fmt.Errorf("invalid digest: %w", err))
			return
		}
		if active[d.Digest] {
			fail(fmt.Errorf("descriptor cycle detected"))
			return
		}
		key := visit{d.Digest, string(d.MediaType), d.Size}
		if seen[key] {
			return
		}
		seen[key] = true
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
			return
		}
		info, statErr := f.Stat()
		closeErr := f.Close()
		if statErr != nil {
			fail(fmt.Errorf("cannot stat cache blob: %w", statErr))
			return
		}
		if closeErr != nil {
			fail(fmt.Errorf("cannot close cache blob: %w", closeErr))
			return
		}
		if !info.Mode().IsRegular() {
			fail(fmt.Errorf("cache blob %s is not a regular file", path))
			return
		}
		if info.Size() != d.Size {
			fail(fmt.Errorf("size mismatch at %s: found %d bytes", path, info.Size()))
			// Still inspect readable JSON to discover other missing children.
		}
		if !d.MediaType.IsIndex() && !d.MediaType.IsImage() && role != "config" {
			return
		}
		raw, err := p.cache.Bytes(d.Digest)
		if err != nil {
			fail(fmt.Errorf("cannot read cache blob: %w", err))
			return
		}
		switch {
		case d.MediaType.IsIndex():
			var index v1.IndexManifest
			if err := json.Unmarshal(raw, &index); err != nil {
				fail(fmt.Errorf("malformed index JSON at %s: %w", path, err))
				return
			}
			if index.SchemaVersion != 2 {
				fail(fmt.Errorf("invalid index schema version %d at %s", index.SchemaVersion, path))
				return
			}
			for i, child := range index.Manifests {
				kind := "descriptor"
				if child.MediaType.IsIndex() {
					kind = "nested index"
				} else if child.MediaType.IsImage() {
					kind = "image manifest"
				}
				walk(child, context, fmt.Sprintf("manifests[%d] %s", i, kind))
			}
		case d.MediaType.IsImage():
			var manifest v1.Manifest
			if err := json.Unmarshal(raw, &manifest); err != nil {
				fail(fmt.Errorf("malformed manifest JSON at %s: %w", path, err))
				return
			}
			if manifest.SchemaVersion != 2 {
				fail(fmt.Errorf("invalid manifest schema version %d at %s", manifest.SchemaVersion, path))
				return
			}
			walk(manifest.Config, context, "config")
			for i, layer := range manifest.Layers {
				walk(layer, context, fmt.Sprintf("layers[%d]", i))
			}
		default:
			if !json.Valid(raw) {
				fail(fmt.Errorf("malformed config JSON at %s", path))
			}
		}
	}
	walk(*desc, "cache reference "+name, "root")
	if len(issues) > 0 {
		return &pushContentError{name: name, issues: issues}
	}
	return nil
}
