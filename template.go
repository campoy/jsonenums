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

// Added as a .go file to avoid embedding issues of the template.

package main

import (
	"strings"
	"text/template"
)

var generatedTmpl = template.Must(template.New("generated").Funcs(template.FuncMap{
	"toLower": strings.ToLower,
	"toUpper": strings.ToUpper,
}).Parse(`
// generated by jsonenums {{.Command}}; DO NOT EDIT

package {{.PackageName}}

import (
    "encoding/json"
    "fmt"
)

{{ $upper := .Upper}}
{{ $lower := .Lower}}
{{ $noStringer := .NoStringer }}
{{range $typename, $values := .TypesAndValues}}

var (
    _{{$typename}}NameToValue = map[string]{{$typename}} {
        {{range $values}}"{{ if $lower }}{{ toLower . }}{{ else if $upper }}{{ toUpper .}}{{ else }}{{.}}{{end}}": {{.}},
        {{end}}
    }

    _{{$typename}}ValueToName = map[{{$typename}}]string {
        {{range $values}}{{.}}: "{{ if $lower }}{{ toLower . }}{{ else if $upper }}{{ toUpper .}}{{ else }}{{.}}{{end}}",
        {{end}}
    }
)

{{ if not $noStringer }}
func init() {
    var v {{$typename}}
    if _, ok := interface{}(v).(fmt.Stringer); ok {
        _{{$typename}}NameToValue = map[string]{{$typename}} {
            {{range $values}}interface{}({{.}}).(fmt.Stringer).String(): {{.}},
            {{end}}
        }
    }
}
{{ end }}

// MarshalJSON is generated so {{$typename}} satisfies json.Marshaler.
func (r {{$typename}}) MarshalJSON() ([]byte, error) {
    if s, ok := interface{}(r).(fmt.Stringer); ok {
        return json.Marshal(s.String())
    }
    s, ok := _{{$typename}}ValueToName[r]
    if !ok {
        return nil, fmt.Errorf("invalid {{$typename}}: %d", r)
    }
    return json.Marshal(s)
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

{{end}}
`))
