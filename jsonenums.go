// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// JSONenums is a tool to automate the creation of methods that satisfy the
// fmt.Stringer, json.Marshaler and json.Unmarshaler interfaces.
// Given the name of a (signed or unsigned) integer type T that has constants
// defined, stringer will create a new self-contained Go source file implementing
//
//  func (t T) String() string
//  func (t T) MarshalJSON() ([]byte, error)
//  func (t *T) UnmarshalJSON([]byte) error
//
// The file is created in the same package and directory as the package that defines T.
// It has helpful defaults designed for use with go generate.
//
// JSONenums is a simple implementation of a concept and the code might not be
// the most performant or beautiful to read.
//
// For example, given this snippet,
//
//	package painkiller
//
//	type Pill int
//
//	const (
//		Placebo Pill = iota
//		Aspirin
//		Ibuprofen
//		Paracetamol
//		Acetaminophen = Paracetamol
//	)
//
// running this command
//
//	jsonenums -type=Pill
//
// in the same directory will create the file pill_jsonenums.go, in package painkiller,
// containing a definition of
//
//  func (r Pill) String() string
//  func (r Pill) MarshalJSON() ([]byte, error)
//  func (r *Pill) UnmarshalJSON([]byte) error
//
// That method will translate the value of a Pill constant to the string representation
// of the respective constant name, so that the call fmt.Print(painkiller.Aspirin) will
// print the string "Aspirin".
//
// Typically this process would be run using go generate, like this:
//
//	//go:generate stringer -type=Pill
//
// If multiple constants have the same value, the lexically first matching name will
// be used (in the example, Acetaminophen will print as "Paracetamol").
//
// With no arguments, it processes the package in the current directory.
// Otherwise, the arguments must name a single directory holding a Go package
// or a set of Go source files that represent a single Go package.
//
// The -type flag accepts a comma-separated list of types so a single run can
// generate methods for multiple types. The default output file is t_string.go,
// where t is the lower-cased name of the first type listed. THe suffix can be
// overridden with the -suffix flag.
//
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/format"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/exact"
	"golang.org/x/tools/go/types"

	_ "golang.org/x/tools/go/gcimporter"
)

var (
	typeNames    = flag.String("type", "", "comma-separated list of type names; must be set")
	outputSuffix = flag.String("suffix", "_jsonenums", "suffix to be added to the output file")
)

func main() {
	flag.Parse()
	if len(*typeNames) == 0 {
		log.Fatalf("the flag -type must be set")
	}
	types := strings.Split(*typeNames, ",")

	// Only one directory at a time can be processed, and the default is ".".
	dir := "."
	if args := flag.Args(); len(args) == 1 {
		dir = args[0]
	} else if len(args) > 1 {
		log.Fatalf("only one directory at a time")
	}

	pkg, err := parsePackage(dir, *outputSuffix+".go")
	if err != nil {
		log.Fatalf("parsing package: %v", err)
	}

	var analysis = struct {
		Command        string
		PackageName    string
		TypesAndValues map[string][]string
	}{
		Command:        strings.Join(os.Args[1:], " "),
		PackageName:    pkg.name,
		TypesAndValues: make(map[string][]string),
	}

	// Run generate for each type.
	for _, typeName := range types {
		values, err := pkg.valuesOfType(typeName)
		if err != nil {
			log.Fatalf("finding values for type %v: %v", typeName, err)
		}
		analysis.TypesAndValues[typeName] = values

		var buf bytes.Buffer
		if err := generatedTmpl.Execute(&buf, analysis); err != nil {
			log.Fatalf("generating code: %v", err)
		}

		src, err := format.Source(buf.Bytes())
		if err != nil {
			// Should never happen, but can arise when developing this code.
			// The user can compile the output to see the error.
			log.Printf("warning: internal error: invalid Go generated: %s", err)
			log.Printf("warning: compile the package to analyze the error")
			src = buf.Bytes()
		}

		output := strings.ToLower(typeName + *outputSuffix + ".go")
		outputPath := filepath.Join(dir, output)
		if err := ioutil.WriteFile(outputPath, src, 0644); err != nil {
			log.Fatalf("writing output: %s", err)
		}
	}
}

type Package struct {
	name  string
	files []*ast.File

	defs map[*ast.Ident]types.Object
}

// parsePackage parses the package in the given directory and returns it.
func parsePackage(directory string, skipSuffix string) (*Package, error) {
	pkgDir, err := build.Default.ImportDir(directory, 0)
	if err != nil {
		return nil, fmt.Errorf("cannot process directory %s: %s", directory, err)
	}

	var files []*ast.File
	fs := token.NewFileSet()
	for _, name := range pkgDir.GoFiles {
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, skipSuffix) {
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
		name:  files[0].Name.Name,
		files: files,
		defs:  defs,
	}, nil
}

// generate produces the String method for the named type.
func (pkg *Package) valuesOfType(typeName string) ([]string, error) {
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
