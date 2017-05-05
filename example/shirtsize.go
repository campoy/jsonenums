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

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
)

type TestCasing int

const (
	caseMadnessA TestCasing = iota
	caseMaDnEEsB
	normalCaseExample
)

//go:generate jsonenums -type=ShirtSize

type ShirtSize byte

const (
	NA ShirtSize = iota
	XS
	S
	M
	L
	XL
)

//go:generate jsonenums -type=WeekDay

type WeekDay int

const (
	Monday WeekDay = iota
	Tuesday
	Wednesday
	Thursday
	Friday
	Saturday
	Sunday
)

func (d WeekDay) String() string {
	switch d {
	case Monday:
		return "Dilluns"
	case Tuesday:
		return "Dimarts"
	case Wednesday:
		return "Dimecres"
	case Thursday:
		return "Dijous"
	case Friday:
		return "Divendres"
	case Saturday:
		return "Dissabte"
	case Sunday:
		return "Diumenge"
	default:
		return "invalid WeekDay"
	}
}

func main() {
	v := struct {
		Size ShirtSize
		Day  WeekDay
	}{M, Friday}
	if err := json.NewEncoder(os.Stdout).Encode(v); err != nil {
		log.Fatal(err)
	}

	input := `{"Size":"XL", "Day":"Dimarts"}`
	if err := json.NewDecoder(strings.NewReader(input)).Decode(&v); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("decoded %s as %+v\n", input, v)
}
