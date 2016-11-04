// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Added as a .go file to avoid embedding issues of the template.

package main

import (
	"strings"
	"text/template"
)

var generatedTmpl = template.Must(template.New("generated").
	Funcs(template.FuncMap{"toLower": strings.ToLower}).Parse(`
// generated by jsonenums {{.Command}}; DO NOT EDIT

package {{.PackageName}}

import (
{{range .Imports}}
    "{{.}}"
{{end}}
    "fmt"
)
{{$funcPrefixes := .FuncPrefixes}}

{{range $typename, $values := .TypesAndValues}}

var (
    _{{$typename}}NameToValue = map[string]{{$typename}} {
        {{range $values}}"{{.}}": {{.}},
        {{end}}
    }

    _{{$typename}}ValueToName = map[{{$typename}}]string {
        {{range $values}}{{.}}: "{{.}}",
        {{end}}
    }
)

func init() {
    var v {{$typename}}
    if _, ok := interface{}(v).(fmt.Stringer); ok {
        _{{$typename}}NameToValue = map[string]{{$typename}} {
            {{range $values}}interface{}({{.}}).(fmt.Stringer).String(): {{.}},
            {{end}}
        }
    }
}

{{ range $_, $funcPrefix := $funcPrefixes}}

// Marshal{{$funcPrefix}} is generated so {{$typename}} satisfies {{$funcPrefix | toLower}}.Marshaler.
func (r {{$typename}}) Marshal{{$funcPrefix}}() ([]byte, error) {
    if s, ok := interface{}(r).(fmt.Stringer); ok {
        return {{$funcPrefix | toLower}}.Marshal(s.String())
    }
    s, ok := _{{$typename}}ValueToName[r]
    if !ok {
        return nil, fmt.Errorf("invalid {{$typename}}: %d", r)
    }
    return {{$funcPrefix | toLower}}.Marshal(s)
}

// Unmarshal{{$funcPrefix}} is generated so {{$typename}} satisfies {{$funcPrefix | toLower}}.Unmarshaler.
func (r *{{$typename}}) Unmarshal{{$funcPrefix}}(data []byte) error {
    var s string
    if err := {{$funcPrefix | toLower}}.Unmarshal(data, &s); err != nil {
        return fmt.Errorf("{{$typename}} should be a string, got %s", data)
    }
    v, ok := _{{$typename}}NameToValue[s]
    if !ok {
        return fmt.Errorf("invalid {{$typename}} %q", s)
    }
    *r = v
    return nil
}

{{end}}

{{end}}
`))
