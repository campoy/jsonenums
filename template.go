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

{{range $_, $funcPrefix := $funcPrefixes}}

{{if eq $funcPrefix "JSON"}}
// MarshalJSON is generated so {{$typename}} satisfies json.Marshaler.
func (r {{$typename}}) MarshalJSON() ([]byte, error) {
    if s, ok := interface{}(r).(fmt.Stringer); ok {
        return json.Marshal(s.String())
    }
    s, ok := _{{$typename}}ValueToName[r]
    if !ok {
        return nil, fmt.Errorf("invalid {{$typename}}: %d", r)
    }
    return {{$funcPrefix | toLower}}.Marshal(s)
}

// UnmarshalJSON is generated so {{$typename}} satisfies json.Unmarshaler.
func (r *{{$typename}}) UnmarshalJSON(data []byte) error {
    var s string
    if err := json.Unmarshal(data, &s); err != nil {
        return fmt.Errorf("{{$typename}} should be a string, got %s", data)
    }
    v, ok := _{{$typename}}NameToValue[s]
    if !ok {
        return fmt.Errorf("invalid {{$typename}} %q", s)
    }
    *r = v
    return nil
}
{{else if eq $funcPrefix "BSON"}}
// GetBSON is generated so {{$typename}} satisfies bson.Marshaler.
func (r {{$typename}}) GetBSON() (interface{}, error) {
        var s string
        if stringer, ok := interface{}(r).(fmt.Stringer); ok {
                s = stringer.String()
        }
        s, ok := _{{$typename}}ValueToName[r]
        if !ok {
                return nil, fmt.Errorf("invalid {{$typename}}: %d", r)
        }
        return s, nil
}

// SetBSON is generated so {{$typename}} satisfies bson.Unmarshaler.
func (r *{{$typename}}) SetBSON(raw bson.Raw) error {
        var s []byte
        if err := raw.Unmarshal(&s); err != nil {
                return fmt.Errorf("{{$typename}} should be a string, got %s", raw)
        }
        v, ok := _{{$typename}}NameToValue[string(s)]
        if !ok {
                return fmt.Errorf("invalid {{$typename}} %q", s)
        }
        *r = v
        return nil
}

{{end}}

{{end}}

{{end}}
`))
