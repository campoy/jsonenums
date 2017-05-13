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

package parser

import (
	"go/build"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func must(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}

var fakeCode = `
package main
import "fmt"
func main() {
	fmt.Println("Hello world!")
}
`

func TestParseFromMultipleGopath(t *testing.T) {
	gopaths := filepath.SplitList(build.Default.GOPATH)
	if len(gopaths) < 2 {
		t.Skipf("No multiple GOPATH (%s) exists, skiping..", build.Default.GOPATH)
	}
	gopath := gopaths[len(gopaths)-1]
	dir := filepath.Join(gopath, "src", "foo")
	defer func() { must(t, os.RemoveAll(dir)) }()
	must(t, os.MkdirAll(dir, 0755))
	must(t, ioutil.WriteFile(filepath.Join(dir, "main.go"), []byte(fakeCode), 0644))

	if _, err := ParsePackage(dir); err != nil {
		t.Fatalf("Parse package (%v): %v", dir, err)
	}
}
