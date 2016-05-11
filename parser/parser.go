// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package parser parses Go code and keeps track of all the types defined
// and provides access to all the constants defined for an int type.
package parser

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/constant"
	"go/token"
	"go/types"
	"log"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/loader"
)

// A Package contains all the information related to a parsed package.
type Package struct {
	Name  string
	files []*ast.File

	defs map[*ast.Ident]types.Object
}

// ParsePackage parses the package in the given directory and returns it.
func ParsePackage(directory string) (*Package, error) {
	relDir, err := filepath.Rel(filepath.Join(build.Default.GOPATH, "src"), directory)
	if err != nil {
		return nil, fmt.Errorf("provided directory not under GOPATH (%s): %v",
			build.Default.GOPATH, err)
	}

	conf := loader.Config{TypeChecker: types.Config{FakeImportC: true}}
	conf.Import(relDir)
	program, err := conf.Load()
	if err != nil {
		return nil, fmt.Errorf("couldn't load package: %v", err)
	}

	pkgInfo := program.Package(relDir)
	return &Package{
		Name:  pkgInfo.Pkg.Name(),
		files: pkgInfo.Files,
		defs:  pkgInfo.Defs,
	}, nil
}

// generate produces the String method for the named type.
func (pkg *Package) ValuesOfType(typeName string) ([]string, error) {
	var values, inspectErrs []string
	for _, file := range pkg.files {
		ast.Inspect(file, func(node ast.Node) bool {
			decl, ok := node.(*ast.GenDecl)
			if !ok || decl.Tok != token.CONST {
				// We only care about const declarations.
				return true
			}

			if vs, err := pkg.valuesOfTypeIn(typeName, decl); err != nil {
				inspectErrs = append(inspectErrs, err.Error())
			} else {
				values = append(values, vs...)
			}
			return false
		})
	}
	if len(inspectErrs) > 0 {
		return nil, fmt.Errorf("inspecting code:\n\t%v", strings.Join(inspectErrs, "\n\t"))
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("no values defined for type %s", typeName)
	}
	return values, nil
}

func (pkg *Package) valuesOfTypeIn(typeName string, decl *ast.GenDecl) ([]string, error) {
	var values []string

	// The name of the type of the constants we are declaring.
	// Can change if this is a multi-element declaration.
	typ := ""
	// Loop over the elements of the declaration. Each element is a ValueSpec:
	// a list of names possibly followed by a type, possibly followed by values.
	// If the type and value are both missing, we carry down the type (and value,
	// but the "go/types" package takes care of that).
	for _, spec := range decl.Specs {
		vspec := spec.(*ast.ValueSpec) // Guaranteed to succeed as this is CONST.
		if vspec.Type == nil && len(vspec.Values) > 0 {
			// "X = 1". With no type but a value, the constant is untyped.
			// Skip this vspec and reset the remembered type.
			typ = ""
			continue
		}
		if vspec.Type != nil {
			// "X T". We have a type. Remember it.
			ident, ok := vspec.Type.(*ast.Ident)
			if !ok {
				continue
			}
			typ = ident.Name
		}
		if typ != typeName {
			// This is not the type we're looking for.
			continue
		}

		// We now have a list of names (from one line of source code) all being
		// declared with the desired type.
		// Grab their names and actual values and store them in f.values.
		for _, name := range vspec.Names {
			if name.Name == "_" {
				continue
			}
			// This dance lets the type checker find the values for us. It's a
			// bit tricky: look up the object declared by the name, find its
			// types.Const, and extract its value.
			obj, ok := pkg.defs[name]
			if !ok {
				return nil, fmt.Errorf("no value for constant %s", name)
			}
			info := obj.Type().Underlying().(*types.Basic).Info()
			if info&types.IsInteger == 0 {
				return nil, fmt.Errorf("can't handle non-integer constant type %s", typ)
			}
			value := obj.(*types.Const).Val() // Guaranteed to succeed as this is CONST.
			if value.Kind() != constant.Int {
				log.Fatalf("can't happen: constant is not an integer %s", name)
			}
			values = append(values, name.Name)
		}
	}
	return values, nil
}
