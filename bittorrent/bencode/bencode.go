// Copyright 2016 Jimmy Zelinskie
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package bencode implements bencoding of data as defined in BEP 3 using
// type assertion over reflection for performance.
package bencode

// Dict represents a bencode dictionary.
type Dict map[string]interface{}

// NewDict allocates the memory for a Dict.
func NewDict() Dict {
	return make(Dict)
}

// List represents a bencode list.
type List []interface{}

// NewList allocates the memory for a List.
func NewList() List {
	return make(List, 0)
}
