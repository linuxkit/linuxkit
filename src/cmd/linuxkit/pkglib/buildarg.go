package pkglib

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	buildArgSpecialPrefix = "@lkt:"
	buildArgPkgPrefix     = "pkg:"
	buildArgsPkgPrefix    = "pkgs:"
	buildArgsKeyStemChar  = "%"
)

// pkgImageName reads just the org and image fields from a package's build.yml
// without computing any hashes. This is used to generate build arg key names
// for @lkt:pkgs: wildcards without risking dependency cycles.
func pkgImageName(pkgPath, buildYML string) (string, error) {
	var pi struct {
		Image string `yaml:"image"`
		Org   string `yaml:"org"`
	}
	pi.Org = "linuxkit" // default
	b, err := os.ReadFile(filepath.Join(pkgPath, buildYML))
	if err != nil {
		return "", fmt.Errorf("reading build yml for %s: %w", pkgPath, err)
	}
	dec := yaml.NewDecoder(bytes.NewReader(b))
	_ = dec.Decode(&pi) // ignore unknown-fields error; we only need image/org
	if pi.Image == "" {
		return "", fmt.Errorf("no image field in build yml for %s", pkgPath)
	}
	return pi.Org + "/" + pi.Image, nil
}

// TransformBuildArgValue transforms a build arg pair whose value starts with
// the special linuxkit prefix. Resolves @lkt:pkg: and @lkt:pkgs: references
// by computing package tags via pkglib.
//
// For hash-dir-aware resolution (to avoid cycles and respect version-specific
// build.yml variants), use transformBuildArgValue with a non-empty hashDir.
func TransformBuildArgValue(line, anchorFile string) ([]string, error) {
	return transformBuildArgValue(line, anchorFile, "", false)
}

// transformBuildArgValue is the internal implementation with optional hash-dir
// support.
//
// When hashDir is non-empty, @lkt: dep references are resolved by reading the
// stored tag from <hashDir>/<pkgname>.hash. This avoids dependency cycles and
// ensures version-specific tags (e.g. build-2.4.yml for ZFS) are used.
//
// If a dep's hash file is missing:
//   - strictDeps=true: error ("run update-hashes first")
//   - strictDeps=false: fall back to NewFromConfig with DefaultPkgBuildYML
func transformBuildArgValue(line, anchorFile, hashDir string, strictDeps bool) ([]string, error) {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid build-arg, must be in format 'arg=value': %s", line)
	}
	key := parts[0]
	val := parts[1]
	if !strings.HasPrefix(val, buildArgSpecialPrefix) {
		return []string{line}, nil
	}
	stripped := strings.TrimPrefix(val, buildArgSpecialPrefix)
	var final []string
	switch {
	case strings.HasPrefix(stripped, buildArgPkgPrefix):
		pkgPath := strings.TrimPrefix(stripped, buildArgPkgPrefix)
		if !strings.HasPrefix(pkgPath, "/") {
			anchorDir, err := filepath.Abs(filepath.Dir(anchorFile))
			if err != nil {
				return nil, fmt.Errorf("error getting absolute path for anchor file %q: %v", anchorFile, err)
			}
			pkgPath = filepath.Clean(filepath.Join(anchorDir, pkgPath))
		}
		tag, err := resolveDepTag(pkgPath, hashDir, strictDeps)
		if err != nil {
			return nil, err
		}
		final = append(final, fmt.Sprintf("%s=%s", key, tag))

	case strings.HasPrefix(stripped, buildArgsPkgPrefix):
		if !strings.Contains(key, buildArgsKeyStemChar) {
			return nil, fmt.Errorf("invalid build arg key %q, must contain a '%s'", key, buildArgsKeyStemChar)
		}
		pkgPath := strings.TrimPrefix(stripped, buildArgsPkgPrefix)
		if !strings.HasPrefix(pkgPath, "/") {
			anchorDir, err := filepath.Abs(filepath.Dir(anchorFile))
			if err != nil {
				return nil, fmt.Errorf("error getting absolute path for anchor file %q: %v", anchorFile, err)
			}
			pkgPath = filepath.Clean(filepath.Join(anchorDir, pkgPath))
		}
		matches, err := filepath.Glob(pkgPath)
		if err != nil {
			return nil, fmt.Errorf("error globbing package path %q: %v", pkgPath, err)
		}
		if len(matches) == 0 {
			return nil, fmt.Errorf("no packages found matching path %q", pkgPath)
		}
		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil {
				return nil, fmt.Errorf("error stating package path %q: %v", match, err)
			}
			if !info.IsDir() || strings.HasPrefix(info.Name(), ".") {
				continue
			}
			tag, err := resolveDepTag(match, hashDir, strictDeps)
			if err != nil {
				return nil, err
			}
			// Get the image name to construct the build arg key.
			// Use pkgImageName (reads only image/org from build.yml) to avoid
			// triggering recursive hash computation and potential cycles.
			imageName, err := pkgImageName(match, DefaultPkgBuildYML)
			if err != nil {
				return nil, fmt.Errorf("getting image name for %s: %w", match, err)
			}
			image := strings.ReplaceAll(imageName, "/", "_")
			image = strings.ReplaceAll(image, "-", "_")
			image = strings.ToUpper(image)
			updatedKey := strings.ReplaceAll(key, buildArgsKeyStemChar, image)
			final = append(final, fmt.Sprintf("%s=%s", updatedKey, tag))
		}
		if len(final) == 0 {
			return nil, fmt.Errorf("no packages found at path %q", pkgPath)
		}
	default:
		return nil, fmt.Errorf("unknown linuxkit build arg value %q", val)
	}
	return final, nil
}

// resolveDepTag returns the tag for the package at pkgPath.
//
// When hashDir is non-empty it first checks <hashDir>/<pkgname>.hash.
// If the hash file exists, its stored tag is returned directly (no recursion).
// If absent and strictDeps is true, an error is returned.
// Otherwise it falls back to computing the tag via NewFromConfig.
func resolveDepTag(pkgPath, hashDir string, strictDeps bool) (string, error) {
	if hashDir != "" {
		m, err := readHashManifest(hashDir, pkgPath)
		if err != nil {
			return "", fmt.Errorf("reading dep hash for %s: %w", pkgPath, err)
		}
		if m != nil && m.Tag != "" {
			return m.Tag, nil
		}
		if strictDeps {
			return "", fmt.Errorf("dep %s has no hash file in %s; run update-hashes first",
				filepath.Base(pkgPath), hashDir)
		}
		// lenient: fall through to NewFromConfig
	}
	pkgs, err := NewFromConfig(PkglibConfig{BuildYML: DefaultPkgBuildYML, HashCommit: DefaultPkgCommit}, pkgPath)
	if err != nil {
		return "", err
	}
	if len(pkgs) == 0 {
		return "", fmt.Errorf("no package found at path %q", pkgPath)
	}
	return pkgs[0].Tag(), nil
}
