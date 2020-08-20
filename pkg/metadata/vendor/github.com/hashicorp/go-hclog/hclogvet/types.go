// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file contains the pieces of the tool that use typechecking from the go/types package.

package main

import (
	"go/ast"
	"go/build"
	"go/importer"
	"go/token"
	"go/types"
	"strings"
)

// stdImporter is the importer we use to import packages.
// It is shared so that all packages are imported by the same importer.
var stdImporter types.Importer

var (
	errorType     *types.Interface
	stringerType  *types.Interface // possibly nil
	formatterType *types.Interface // possibly nil
)

func inittypes() {
	errorType = types.Universe.Lookup("error").Type().Underlying().(*types.Interface)

	if typ := importType("fmt", "Stringer"); typ != nil {
		stringerType = typ.Underlying().(*types.Interface)
	}
	if typ := importType("fmt", "Formatter"); typ != nil {
		formatterType = typ.Underlying().(*types.Interface)
	}
}

// isNamedType reports whether t is the named type path.name.
func isNamedType(t types.Type, path, name string) bool {
	n, ok := t.(*types.Named)
	if !ok {
		return false
	}
	obj := n.Obj()
	return obj.Name() == name && isPackage(obj.Pkg(), path)
}

// isPackage reports whether pkg has path as the canonical path,
// taking into account vendoring effects
func isPackage(pkg *types.Package, path string) bool {
	if pkg == nil {
		return false
	}

	return pkg.Path() == path ||
		strings.HasSuffix(pkg.Path(), "/vendor/"+path)
}

// importType returns the type denoted by the qualified identifier
// path.name, and adds the respective package to the imports map
// as a side effect. In case of an error, importType returns nil.
func importType(path, name string) types.Type {
	pkg, err := stdImporter.Import(path)
	if err != nil {
		// This can happen if the package at path hasn't been compiled yet.
		warnf("import failed: %v", err)
		return nil
	}
	if obj, ok := pkg.Scope().Lookup(name).(*types.TypeName); ok {
		return obj.Type()
	}
	warnf("invalid type name %q", name)
	return nil
}

func (pkg *Package) check(fs *token.FileSet, astFiles []*ast.File) []error {
	if stdImporter == nil {
		if *source {
			stdImporter = importer.For("source", nil)
		} else {
			stdImporter = importer.Default()
		}
		inittypes()
	}
	pkg.defs = make(map[*ast.Ident]types.Object)
	pkg.uses = make(map[*ast.Ident]types.Object)
	pkg.selectors = make(map[*ast.SelectorExpr]*types.Selection)
	pkg.spans = make(map[types.Object]Span)
	pkg.types = make(map[ast.Expr]types.TypeAndValue)

	var allErrors []error
	config := types.Config{
		// We use the same importer for all imports to ensure that
		// everybody sees identical packages for the given paths.
		Importer: stdImporter,
		// By providing a Config with our own error function, it will continue
		// past the first error. We collect them all for printing later.
		Error: func(e error) {
			allErrors = append(allErrors, e)
		},

		Sizes: archSizes,
	}
	info := &types.Info{
		Selections: pkg.selectors,
		Types:      pkg.types,
		Defs:       pkg.defs,
		Uses:       pkg.uses,
	}
	typesPkg, err := config.Check(pkg.path, fs, astFiles, info)
	if len(allErrors) == 0 && err != nil {
		allErrors = append(allErrors, err)
	}
	pkg.typesPkg = typesPkg
	// update spans
	for id, obj := range pkg.defs {
		pkg.growSpan(id, obj)
	}
	for id, obj := range pkg.uses {
		pkg.growSpan(id, obj)
	}
	return allErrors
}

func isConvertibleToString(typ types.Type) bool {
	if bt, ok := typ.(*types.Basic); ok && bt.Kind() == types.UntypedNil {
		// We explicitly don't want untyped nil, which is
		// convertible to both of the interfaces below, as it
		// would just panic anyway.
		return false
	}
	if types.ConvertibleTo(typ, errorType) {
		return true // via .Error()
	}
	if stringerType != nil && types.ConvertibleTo(typ, stringerType) {
		return true // via .String()
	}
	return false
}

// hasBasicType reports whether x's type is a types.Basic with the given kind.
func (f *File) hasBasicType(x ast.Expr, kind types.BasicKind) bool {
	t := f.pkg.types[x].Type
	if t != nil {
		t = t.Underlying()
	}
	b, ok := t.(*types.Basic)
	return ok && b.Kind() == kind
}

// hasMethod reports whether the type contains a method with the given name.
// It is part of the workaround for Formatters and should be deleted when
// that workaround is no longer necessary.
// TODO: This could be better once issue 6259 is fixed.
func (f *File) hasMethod(typ types.Type, name string) bool {
	// assume we have an addressable variable of type typ
	obj, _, _ := types.LookupFieldOrMethod(typ, true, f.pkg.typesPkg, name)
	_, ok := obj.(*types.Func)
	return ok
}

var archSizes = types.SizesFor("gc", build.Default.GOARCH)
