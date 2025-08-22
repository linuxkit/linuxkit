package pkglib

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	buildArgSpecialPrefix = "@lkt:"
	buildArgPkgPrefix     = "pkg:"
	buildArgsPkgPrefix    = "pkgs:"
	buildArgsKeyStemChar  = "%"
)

// TransformBuildArgValue transforms a build arg pair whose value starts with the special linuxkit prefix.
func TransformBuildArgValue(line, anchorFile string) ([]string, error) {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid build-arg, must be in format 'arg=value': %s", line)
	}
	key := parts[0]
	val := parts[1]
	// check if the value is a special linuxkit value
	if !strings.HasPrefix(val, buildArgSpecialPrefix) {
		return []string{line}, nil
	}
	stripped := strings.TrimPrefix(val, buildArgSpecialPrefix)
	var final []string
	// see if we know what kind of value it is
	switch {
	case strings.HasPrefix(stripped, buildArgPkgPrefix):
		pkgPath := strings.TrimPrefix(stripped, buildArgPkgPrefix)
		// see if it is an absolute or relative path
		if !strings.HasPrefix(pkgPath, "/") {
			anchorDir, err := filepath.Abs(filepath.Dir(anchorFile))
			if err != nil {
				return nil, fmt.Errorf("error getting absolute path for anchor file %q: %v", anchorFile, err)
			}
			pkgPath = filepath.Clean(filepath.Join(anchorDir, pkgPath))
		}
		pkgs, err := NewFromConfig(PkglibConfig{BuildYML: DefaultPkgBuildYML, HashCommit: DefaultPkgCommit}, pkgPath)
		if err != nil {
			return nil, err
		}
		if len(pkgs) == 0 {
			return nil, fmt.Errorf("no package found at path %q", pkgPath)
		}
		p := pkgs[0]
		tag := p.Tag()
		final = append(final, fmt.Sprintf("%s=%s", key, tag))
	case strings.HasPrefix(stripped, buildArgsPkgPrefix):
		// validate the key
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
		var finalMatches []string
		for _, match := range matches {
			// ensure the match is a directory
			info, err := os.Stat(match)
			if err != nil {
				return nil, fmt.Errorf("error stating package path %q: %v", match, err)
			}
			if !info.IsDir() {
				continue
			}
			if strings.HasPrefix(info.Name(), ".") {
				continue
			}
			finalMatches = append(finalMatches, match)
		}
		pkgs, err := NewFromConfig(PkglibConfig{BuildYML: DefaultPkgBuildYML, HashCommit: DefaultPkgCommit}, finalMatches...)
		if err != nil {
			return nil, fmt.Errorf("error loading packages from paths %q: %v", pkgPath, err)
		}
		if len(pkgs) == 0 {
			return nil, fmt.Errorf("no packages found at path %q", pkgPath)
		}
		for _, p := range pkgs {
			tag := p.Tag()
			// generate the special build arg key
			image := strings.ReplaceAll(p.OrgImage(), "/", "_")
			image = strings.ReplaceAll(image, "-", "_")
			image = strings.ToUpper(image)
			updatedKey := strings.ReplaceAll(key, buildArgsKeyStemChar, image)
			final = append(final, fmt.Sprintf("%s=%s", updatedKey, tag))
		}
	default:
		// something unknown
		return nil, fmt.Errorf("unknown linuxkit build arg value %q", val)
	}
	return final, nil
}
