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
	"go/parser"
	"go/token"
	"log"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/exact"
	_ "golang.org/x/tools/go/gcimporter"
	"golang.org/x/tools/go/types"
)

// A Package contains all the information related to a parsed package.
type Package struct {
	Name  string
	files []*ast.File

	defs map[*ast.Ident]types.Object
}

// ParsePackage parses the package in the given directory and returns it.
func ParsePackage(directory, skipPrefix, skipSuffix string) (*Package, error) {
	pkgDir, err := build.Default.ImportDir(directory, 0)
	if err != nil {
		return nil, fmt.Errorf("cannot process directory %s: %s", directory, err)
	}

	var files []*ast.File
	fs := token.NewFileSet()
	for _, name := range pkgDir.GoFiles {
		if !strings.HasSuffix(name, ".go") ||
			(skipSuffix != "" && strings.HasPrefix(name, skipPrefix) &&
				strings.HasSuffix(name, skipSuffix)) {
			continue
		}
		if directory != "." {
			name = filepath.Join(directory, name)
		}
		f, err := parser.ParseFile(fs, name, nil, 0)
		if err != nil {
			return nil, fmt.Errorf("parsing file %v: %v", name, err)
		}
		files = append(files, f)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("%s: no buildable Go files", directory)
	}

	// type-check the package
	defs := make(map[*ast.Ident]types.Object)
	config := types.Config{FakeImportC: true}
	info := &types.Info{Defs: defs}
	if _, err := config.Check(directory, fs, files, info); err != nil {
		return nil, fmt.Errorf("type-checking package: %v", err)
	}

	return &Package{
		Name:  files[0].Name.Name,
		files: files,
		defs:  defs,
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
			if value.Kind() != exact.Int {
				log.Fatalf("can't happen: constant is not an integer %s", name)
			}
			values = append(values, name.Name)
		}
	}
	return values, nil
}
