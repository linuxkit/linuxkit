package pkglib

import (
	"fmt"
	"path/filepath"
	"strings"
)

const (
	buildArgSpecialPrefix = "@lkt:"
	buildArgPkgPrefix     = "pkg:"
)

// TransformBuildArgValue transforms a build arg pair whose value starts with the special linuxkit prefix.
func TransformBuildArgValue(line, anchorFile string) (string, error) {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid build-arg, must be in format 'arg=value': %s", line)
	}
	key := parts[0]
	val := parts[1]
	// check if the value is a special linuxkit value
	if !strings.HasPrefix(val, buildArgSpecialPrefix) {
		return line, nil
	}
	stripped := strings.TrimPrefix(val, buildArgSpecialPrefix)
	var final string
	// see if we know what kind of value it is
	switch {
	case strings.HasPrefix(stripped, buildArgPkgPrefix):
		pkgPath := strings.TrimPrefix(stripped, buildArgPkgPrefix)
		// see if it is an absolute or relative path
		if !strings.HasPrefix(pkgPath, "/") {
			anchorDir, err := filepath.Abs(filepath.Dir(anchorFile))
			if err != nil {
				return "", fmt.Errorf("error getting absolute path for anchor file %q: %v", anchorFile, err)
			}
			pkgPath = filepath.Clean(filepath.Join(anchorDir, pkgPath))
		}
		pkgs, err := NewFromConfig(PkglibConfig{BuildYML: DefaultPkgBuildYML, HashCommit: DefaultPkgCommit}, pkgPath)
		if err != nil {
			return "", err
		}
		if len(pkgs) == 0 {
			return "", fmt.Errorf("no package found at path %q", pkgPath)
		}
		p := pkgs[0]
		tag := p.Tag()
		final = tag
	default:
		// something unknown
		return "", fmt.Errorf("unknown linuxkit build arg value %q", val)
	}
	return fmt.Sprintf("%s=%s", key, final), nil
}
