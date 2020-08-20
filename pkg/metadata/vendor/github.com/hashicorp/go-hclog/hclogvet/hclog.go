// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file contains the printf-checker.

package main

import (
	"go/ast"
	"go/types"
)

func init() {
	register("hclog",
		"check hclog invocations",
		checkHCLog,
		callExpr)
}

var checkHCLogFunc = map[string]bool{
	"Trace": true,
	"Debug": true,
	"Info":  true,
	"Warn":  true,
	"Error": true,
}

func checkHCLog(f *File, node ast.Node) {
	call := node.(*ast.CallExpr)
	fun, _ := call.Fun.(*ast.SelectorExpr)
	typ := f.pkg.types[fun]
	sig, _ := typ.Type.(*types.Signature)
	if sig == nil {
		return // the call is not on of the form x.f()
	}

	recv := f.pkg.types[fun.X]

	if recv.Type == nil {
		return
	}

	if !isNamedType(recv.Type, "github.com/hashicorp/go-hclog", "Logger") {
		return
	}

	if _, ok := checkHCLogFunc[fun.Sel.Name]; !ok {
		return
	}

	if len(call.Args)%2 != 1 {
		f.Badf(call.Pos(), "invalid number of log arguments to %s (%d)", fun.Sel.Name, len(call.Args))
	}
}
