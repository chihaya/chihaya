// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

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
