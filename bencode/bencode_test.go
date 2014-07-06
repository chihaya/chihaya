// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package bencode

import (
	"testing"
	"time"
)

var scalarTests = map[interface{}]string{
	int(42):    "i42e",
	int(-42):   "i-42e",
	uint(42):   "i42e",
	int64(42):  "i42e",
	uint64(42): "i42e",

	"example":        "7:example",
	30 * time.Minute: "i1800e",
}

func TestScalar(t *testing.T) {
	for val, expected := range scalarTests {
		got, err := Marshal(val)
		if err != nil {
			t.Error(err)
		} else if string(got) != expected {
			t.Errorf("\ngot:      %s\nexpected: %s", got, expected)
		}
	}
}
