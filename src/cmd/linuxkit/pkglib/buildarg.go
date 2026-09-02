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

// PkgImageName reads just the org and image fields from a package's build.yml
// without computing any hashes. This is used to generate build arg key names
// for @lkt:pkgs: wildcards without risking dependency cycles.
func PkgImageName(pkgPath, buildYML string) (string, error) {
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

// pkgImageName is an alias for PkgImageName for backward compatibility within the package.
func pkgImageName(pkgPath, buildYML string) (string, error) {
	return PkgImageName(pkgPath, buildYML)
}

// DockerfileARGNames returns the set of ARG names declared in the Dockerfile
// at path. Returns empty map if file cannot be read (permissive fallback:
// when the Dockerfile is unreadable, all packages pass the filter).
func DockerfileARGNames(dockerfilePath string) map[string]bool {
	result := make(map[string]bool)
	data, err := os.ReadFile(dockerfilePath)
	if err != nil {
		return result // permissive: fall back to old behavior (include all)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(strings.ToUpper(line), "ARG ") {
			continue
		}
		name := strings.TrimSpace(line[4:])
		if idx := strings.IndexByte(name, '='); idx >= 0 {
			name = name[:idx]
		}
		result[strings.TrimSpace(name)] = true
	}
	return result
}

// dockerfileARGNames is an alias for DockerfileARGNames for use within the package.
func dockerfileARGNames(dockerfilePath string) map[string]bool {
	return DockerfileARGNames(dockerfilePath)
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
		if tag == "" {
			break // skip — dep couldn't be resolved (lenient mode, no build.yml)
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

		// Load Dockerfile ARGs of the anchor package for filtering.
		// Only packages whose computed ARG key appears in the Dockerfile are included.
		// If the Dockerfile is unreadable, usedARGs is empty → all packages pass (permissive).
		anchorDir := filepath.Dir(anchorFile)
		usedARGs := dockerfileARGNames(filepath.Join(anchorDir, "Dockerfile"))

		dirCount := 0
		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil {
				return nil, fmt.Errorf("error stating package path %q: %v", match, err)
			}
			if !info.IsDir() || strings.HasPrefix(info.Name(), ".") {
				continue
			}
			dirCount++

			// Use hash manifest's stored build-yml to resolve image names for versioned
			// packages (e.g. pkg/zfs has build-2.3.yml, not build.yml).
			buildYMLForImage := DefaultPkgBuildYML
			if hashDir != "" {
				if m, _ := readHashManifest(hashDir, match); m != nil && m.BuildYML != "" {
					buildYMLForImage = m.BuildYML
				}
			}
			imageName, err := pkgImageName(match, buildYMLForImage)
			if err != nil {
				// Skip packages without a usable build yml.
				continue
			}
			image := strings.ReplaceAll(imageName, "/", "_")
			image = strings.ReplaceAll(image, "-", "_")
			image = strings.ToUpper(image)
			updatedKey := strings.ReplaceAll(key, buildArgsKeyStemChar, image)

			// Filter: only include this dep if its ARG is declared in the Dockerfile.
			if len(usedARGs) > 0 && !usedARGs[updatedKey] {
				continue
			}

			tag, err := resolveDepTag(match, hashDir, strictDeps)
			if err != nil {
				return nil, err
			}
			if tag == "" {
				continue // skip — dep couldn't be resolved (lenient mode, no build.yml)
			}
			final = append(final, fmt.Sprintf("%s=%s", updatedKey, tag))
		}
		if dirCount == 0 {
			return nil, fmt.Errorf("no packages found at path %q", pkgPath)
		}
		// len(final) == 0 is OK when Dockerfile filtering excluded all packages.
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
// Otherwise it falls back to computing the tag via NewFromConfig — but only
// when the default build.yml exists. Packages without a default build.yml
// (e.g. versioned packages like pkg/zfs that have only build-2.x.yml) are
// skipped in lenient mode or error in strict mode.
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
		// lenient: fall through to NewFromConfig, but only if build.yml exists
	}
	// Check build.yml exists before attempting to load — versioned packages
	// (e.g. pkg/zfs) have no default build.yml and must be accessed via hash files.
	defaultYML := filepath.Join(pkgPath, DefaultPkgBuildYML)
	if _, statErr := os.Stat(defaultYML); os.IsNotExist(statErr) {
		if strictDeps {
			return "", fmt.Errorf("dep %s has no hash file and no default build.yml; run update-hashes first",
				filepath.Base(pkgPath))
		}
		return "", nil // skip — can't compute tag without a build yml
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
