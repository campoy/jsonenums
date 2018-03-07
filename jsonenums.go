// Copyright 2017 Google Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to writing, software distributed
// under the License is distributed on a "AS IS" BASIS, WITHOUT WARRANTIES OR
// CONDITIONS OF ANY KIND, either express or implied.
//
// See the License for the specific language governing permissions and
// limitations under the License.

// JSONenums is a tool to automate the creation of methods that satisfy the
// fmt.Stringer, json.Marshaler and json.Unmarshaler interfaces.
// Given the name of a (signed or unsigned) integer type T that has constants
// defined, jsonenums will create a new self-contained Go source file implementing
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
//	//go:generate jsonenums -type=Pill
//
// If multiple constants have the same value, the lexically first matching name will
// be used (in the example, Acetaminophen will print as "Paracetamol").
//
// With no arguments, it processes the package in the current directory.
// Otherwise, the arguments must name a single directory holding a Go package
// or a set of Go source files that represent a single Go package.
//
// The -type flag accepts a comma-separated list of types so a single run can
// generate methods for multiple types. The default output file is
// t_jsonenums.go, where t is the lower-cased name of the first type listed.
// The suffix can be overridden with the -suffix flag and a prefix may be added
// with the -prefix flag.
//
package main

import (
	"bytes"
	"flag"
	"go/format"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/campoy/jsonenums/parser"
)

var (
	typeNames        = flag.String("type", "", "comma-separated list of type names; must be set")
	outputLower      = flag.Bool("lower", false, "set the json enum values to lower case")
	outputUpper      = flag.Bool("upper", false, "set the json enum values to upper case")
	outputNoStringer = flag.Bool("no-stringer", false, "disable the usage of stringer if exists")
	outputPrefix     = flag.String("prefix", "", "prefix to be added to the output file")
	outputSuffix     = flag.String("suffix", "_jsonenums", "suffix to be added to the output file")
)

func main() {
	flag.Parse()
	if len(*typeNames) == 0 {
		log.Fatalf("the flag -type must be set")
	}
	types := strings.Split(*typeNames, ",")

	if *outputLower && *outputUpper {
		log.Fatal("cannot -upper and -lower in the same time")
	}

	// Only one directory at a time can be processed, and the default is ".".
	dir := "."
	if args := flag.Args(); len(args) == 1 {
		dir = args[0]
	} else if len(args) > 1 {
		log.Fatalf("only one directory at a time")
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		log.Fatalf("unable to determine absolute filepath for requested path %s: %v",
			dir, err)
	}

	pkg, err := parser.ParsePackage(dir)
	if err != nil {
		log.Fatalf("parsing package: %v", err)
	}

	var analysis = struct {
		Command        string
		PackageName    string
		TypesAndValues map[string][]string
		Lower          bool
		Upper          bool
		NoStringer     bool
	}{
		Command:        strings.Join(os.Args[1:], " "),
		PackageName:    pkg.Name,
		TypesAndValues: make(map[string][]string),
		Lower:          *outputLower,
		Upper:          *outputUpper,
		NoStringer:     *outputNoStringer,
	}

	// Run generate for each type.
	for _, typeName := range types {
		values, err := pkg.ValuesOfType(typeName)
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

		output := strings.ToLower(*outputPrefix + typeName +
			*outputSuffix + ".go")
		outputPath := filepath.Join(dir, output)
		if err := ioutil.WriteFile(outputPath, src, 0644); err != nil {
			log.Fatalf("writing output: %s", err)
		}
	}
}
