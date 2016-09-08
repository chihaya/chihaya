// Package bencode implements bencoding of data as defined in BEP 3 using
// type assertion over reflection for performance.
package bencode

import "bytes"

// Enforce that Dict implements the Marshaler interface.
var _ Marshaler = Dict{}

// Dict represents a bencode dictionary.
type Dict map[string]interface{}

// NewDict allocates the memory for a Dict.
func NewDict() Dict {
	return make(Dict)
}

// MarshalBencode implements the Marshaler interface for Dict.
func (d Dict) MarshalBencode() ([]byte, error) {
	var buf bytes.Buffer
	err := marshalMap(&buf, map[string]interface{}(d))
	return buf.Bytes(), err
}

// Enforce that List implements the Marshaler interface.
var _ Marshaler = List{}

// List represents a bencode list.
type List []interface{}

// MarshalBencode implements the Marshaler interface for List.
func (l List) MarshalBencode() ([]byte, error) {
	var buf bytes.Buffer
	err := marshalList(&buf, []interface{}(l))
	return buf.Bytes(), err
}

// NewList allocates the memory for a List.
func NewList() List {
	return make(List, 0)
}
