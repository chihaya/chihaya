package bencode

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var marshalTests = []struct {
	input    interface{}
	expected []string
}{
	{int(42), []string{"i42e"}},
	{int(-42), []string{"i-42e"}},
	{uint(43), []string{"i43e"}},
	{int64(44), []string{"i44e"}},
	{uint64(45), []string{"i45e"}},
	{int16(44), []string{"i44e"}},
	{uint16(45), []string{"i45e"}},

	{"example", []string{"7:example"}},
	{[]byte("example"), []string{"7:example"}},
	{30 * time.Minute, []string{"i1800e"}},

	{[]string{"one", "two"}, []string{"l3:one3:twoe", "l3:two3:onee"}},
	{[]interface{}{"one", "two"}, []string{"l3:one3:twoe", "l3:two3:onee"}},
	{[]string{}, []string{"le"}},

	{map[string]interface{}{"one": "aa", "two": "bb"}, []string{"d3:one2:aa3:two2:bbe", "d3:two2:bb3:one2:aae"}},
	{map[string]interface{}{}, []string{"de"}},
}

func TestMarshal(t *testing.T) {
	for _, test := range marshalTests {
		got, err := Marshal(test.input)
		assert.Nil(t, err, "marshal should not fail")
		assert.Contains(t, test.expected, string(got), "the marshaled result should be one of the expected permutations")
	}
}

func BenchmarkMarshalScalar(b *testing.B) {
	buf := &bytes.Buffer{}
	encoder := NewEncoder(buf)

	for i := 0; i < b.N; i++ {
		encoder.Encode("test")
		encoder.Encode(123)
	}
}

func BenchmarkMarshalLarge(b *testing.B) {
	data := map[string]interface{}{
		"k1": []string{"a", "b", "c"},
		"k2": 42,
		"k3": "val",
		"k4": uint(42),
	}

	buf := &bytes.Buffer{}
	encoder := NewEncoder(buf)

	for i := 0; i < b.N; i++ {
		encoder.Encode(data)
	}
}
