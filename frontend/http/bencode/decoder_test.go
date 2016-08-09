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

package bencode

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var unmarshalTests = []struct {
	input    string
	expected interface{}
}{
	{"i42e", int64(42)},
	{"i-42e", int64(-42)},

	{"7:example", "example"},

	{"l3:one3:twoe", List{"one", "two"}},
	{"le", List{}},

	{"d3:one2:aa3:two2:bbe", Dict{"one": "aa", "two": "bb"}},
	{"de", Dict{}},
}

func TestUnmarshal(t *testing.T) {
	for _, tt := range unmarshalTests {
		got, err := Unmarshal([]byte(tt.input))
		assert.Nil(t, err, "unmarshal should not fail")
		assert.Equal(t, got, tt.expected, "unmarshalled values should match the expected results")
	}
}

type bufferLoop struct {
	val string
}

func (r *bufferLoop) Read(b []byte) (int, error) {
	n := copy(b, r.val)
	return n, nil
}

func BenchmarkUnmarshalScalar(b *testing.B) {
	d1 := NewDecoder(&bufferLoop{"7:example"})
	d2 := NewDecoder(&bufferLoop{"i42e"})

	for i := 0; i < b.N; i++ {
		d1.Decode()
		d2.Decode()
	}
}

func TestUnmarshalLarge(t *testing.T) {
	data := Dict{
		"k1": List{"a", "b", "c"},
		"k2": int64(42),
		"k3": "val",
		"k4": int64(-42),
	}

	buf, _ := Marshal(data)
	dec := NewDecoder(&bufferLoop{string(buf)})

	got, err := dec.Decode()
	assert.Nil(t, err, "decode should not fail")
	assert.Equal(t, got, data, "encoding and decoding should equal the original value")
}

func BenchmarkUnmarshalLarge(b *testing.B) {
	data := map[string]interface{}{
		"k1": []string{"a", "b", "c"},
		"k2": 42,
		"k3": "val",
		"k4": uint(42),
	}

	buf, _ := Marshal(data)
	dec := NewDecoder(&bufferLoop{string(buf)})

	for i := 0; i < b.N; i++ {
		dec.Decode()
	}
}
