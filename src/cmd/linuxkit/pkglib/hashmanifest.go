package pkglib

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// HashManifest is the YAML structure stored in <hash-dir>/<pkgname>.hash files.
// It records the computed tag for a package along with the build yml used,
// target architecture, and (optionally) the dep tags consumed during the build.
type HashManifest struct {
	Tag      string     `yaml:"tag"`
	BuildYML string     `yaml:"build-yml"`
	Arch     string     `yaml:"arch,omitempty"`
	Deps     []DepEntry `yaml:"deps,omitempty"`
}

// DepEntry records a single dependency consumed during a package build.
type DepEntry struct {
	Path string `yaml:"path"`
	Tag  string `yaml:"tag"`
}

// manifestMatch returns true when the existing manifest matches m on all
// fields that affect whether a rebuild is needed: tag, build-yml, and arch.
func manifestMatch(existing, m HashManifest) bool {
	if existing.Tag != m.Tag {
		return false
	}
	if existing.BuildYML != m.BuildYML {
		return false
	}
	// Arch "" (legacy/unset) matches anything - do not invalidate old files
	// that predate the arch field.
	if m.Arch != "" && existing.Arch != "" && existing.Arch != m.Arch {
		return false
	}
	return true
}

// readHashManifest reads the stored manifest for pkgPath from hashDir.
// Returns (nil, nil) when the file does not exist.
func readHashManifest(hashDir, pkgPath string) (*HashManifest, error) {
	if hashDir == "" {
		return nil, nil
	}
	name := filepath.Base(pkgPath)
	hashFile := filepath.Join(hashDir, name+".hash")
	b, err := os.ReadFile(hashFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading hash file %s: %w", hashFile, err)
	}
	var m HashManifest
	if err := yaml.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("parsing hash file %s: %w", hashFile, err)
	}
	return &m, nil
}

// WriteHashManifest writes m to <hashDir>/<pkgname>.hash using
// write-if-changed semantics.
//
// If the file already exists and matches on tag, build-yml, and arch,
// it is left untouched (mtime preserved) so make does not cascade rebuilds.
//
// When the file is new or any field changed, the content is written and the
// mtime is set to the Unix epoch. This ensures make sees the hash file as
// older than any source file, causing the build recipe to fire.
// After a successful pkg build, the Makefile recipe touches the file to
// set a real mtime, completing the build stamp cycle.
//
// Returns true when the file was actually written.
func WriteHashManifest(hashDir, pkgPath string, m HashManifest) (bool, error) {
	return writeHashManifest(hashDir, pkgPath, m)
}

func writeHashManifest(hashDir, pkgPath string, m HashManifest) (bool, error) {
	if err := os.MkdirAll(hashDir, 0755); err != nil {
		return false, fmt.Errorf("mkdir %s: %w", hashDir, err)
	}
	name := filepath.Base(pkgPath)
	hashFile := filepath.Join(hashDir, name+".hash")

	newBytes, err := yaml.Marshal(&m)
	if err != nil {
		return false, fmt.Errorf("marshalling hash manifest: %w", err)
	}

	if existing, err := os.ReadFile(hashFile); err == nil {
		var existingManifest HashManifest
		if yaml.Unmarshal(existing, &existingManifest) == nil && manifestMatch(existingManifest, m) {
			return false, nil
		}
	}

	if err := os.WriteFile(hashFile, newBytes, 0644); err != nil {
		return false, fmt.Errorf("writing hash file %s: %w", hashFile, err)
	}
	// Set mtime to epoch so make sees the file as stale (older than any
	// source file). The Makefile recipe will touch the file after a
	// successful build, giving it a real timestamp.
	epoch := time.Unix(0, 0)
	if err := os.Chtimes(hashFile, epoch, epoch); err != nil {
		return true, fmt.Errorf("setting mtime on %s: %w", hashFile, err)
	}
	return true, nil
}
