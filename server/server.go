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

// Server is an http server that provides an alternative way of generating code
// based on int types and the constants defined with it.
//
// Use the http flag to change the address on which the server will listen for
// requests. The default is 127.0.0.1:8080.
package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"text/template"

	"go/format"

	"github.com/campoy/jsonenums/parser"
)

func init() {
	http.Handle("/generate", handler(generateHandler))
}

func generateHandler(w http.ResponseWriter, r *http.Request) error {
	if r.Method != "GET" {
		return codeError{fmt.Errorf("only GET accepted"), http.StatusMethodNotAllowed}
	}
	code := r.FormValue("code")
	if code == "" {
		return codeError{fmt.Errorf("no code to be parsed"), http.StatusBadRequest}
	}
	typ := r.FormValue("type")
	if typ == "" {
		return codeError{fmt.Errorf("no type to be analyzed"), http.StatusBadRequest}
	}

	dir, err := createDir(code)
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	pkg, err := parser.ParsePackage(dir)
	if err != nil {
		return fmt.Errorf("parse package: %v", err)
	}

	values, err := pkg.ValuesOfType(typ)
	if err != nil {
		return fmt.Errorf("find values: %v", err)
	}

	t, err := template.New("code").Parse(r.FormValue("template"))
	if err != nil {
		return codeError{fmt.Errorf("parse template: %v", err), http.StatusBadRequest}
	}
	var data = struct {
		PackageName string
		TypeName    string
		Values      []string
	}{pkg.Name, typ, values}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return codeError{fmt.Errorf("execute template: %v", err), http.StatusBadRequest}
	}
	src, err := format.Source(buf.Bytes())
	if err != nil {
		return codeError{fmt.Errorf("code generated is not valid: %v\n%v", err, buf.String()), http.StatusBadRequest}
	}
	w.Write(src)
	return nil
}

func createDir(content string) (string, error) {
	dir, err := ioutil.TempDir("", "jsonenums")
	if err != nil {
		return "", fmt.Errorf("create tmp dir: %v", err)
	}
	f, err := os.Create(filepath.Join(dir, "main.go"))
	if err != nil {
		os.RemoveAll(dir)
		return "", fmt.Errorf("create tmp file: %v", err)
	}
	f.WriteString(content)
	f.Close()
	return dir, err
}

type handler func(http.ResponseWriter, *http.Request) error

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := h(w, r)
	if err != nil {
		code := http.StatusInternalServerError
		if cErr, ok := err.(codeError); ok {
			code = cErr.code
		} else {
			log.Printf("%v: %v", r.URL, code)
		}
		http.Error(w, err.Error(), code)
	}
}

type codeError struct {
	error
	code int
}
