package main

import (
	"fmt"
	"path"
	"strings"

	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/pkglib"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/spec"
)

const (
	templateFlag = "@"
	templatePkg  = "pkg:"
)

func createPackageResolver(baseDir string) spec.PackageResolver {
	return func(pkgTmpl string) (tag string, err error) {
		var pkgValue string
		switch {
		case len(pkgTmpl) == 0, pkgTmpl[0:1] != templateFlag:
			pkgValue = pkgTmpl
		case strings.HasPrefix(pkgTmpl, templateFlag+templatePkg):
			pkgPath := strings.TrimPrefix(pkgTmpl, templateFlag+templatePkg)

			var pkgs []pkglib.Pkg
			pkgConfig := pkglib.PkglibConfig{
				BuildYML:   defaultPkgBuildYML,
				HashCommit: defaultPkgCommit,
				Tag:        defaultPkgTag,
			}
			pkgs, err = pkglib.NewFromConfig(pkgConfig, path.Join(baseDir, pkgPath))
			if err != nil {
				return tag, err
			}
			if len(pkgs) == 0 {
				return tag, fmt.Errorf("no packages found")
			}
			if len(pkgs) > 1 {
				return tag, fmt.Errorf("multiple packages found")
			}
			pkgValue = pkgs[0].FullTag()
		}
		return pkgValue, nil
	}
}
