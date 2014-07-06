// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package bencode

import (
	"testing"
	"time"
)

var tests = []struct {
	input    interface{}
	expected string
}{
	{int(42), "i42e"},
	{int(-42), "i-42e"},
	{uint(43), "i43e"},
	{int64(44), "i44e"},
	{uint64(45), "i45e"},

	{"example", "7:example"},
	{30 * time.Minute, "i1800e"},

	{[]string{"one", "two"}, "l3:one3:twoe"},
	{[]string{}, "le"},

	{
		map[string]interface{}{
			"one": "aa",
			"two": "bb",
		},
		"d3:one2:aa3:two2:bbe",
	},
	{map[string]interface{}{}, "de"},
}

func TestMarshal(t *testing.T) {
	for _, test := range tests {
		got, err := Marshal(test.input)
		if err != nil {
			t.Error(err)
		} else if string(got) != test.expected {
			t.Errorf("\ngot:      %s\nexpected: %s", got, test.expected)
		}
	}
}
