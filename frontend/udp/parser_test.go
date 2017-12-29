package udp

import (
	"fmt"
	"testing"
)

var table = []struct {
	data   []byte
	values map[string]string
	err    error
}{
	{
		[]byte{0x2, 0x5, '/', '?', 'a', '=', 'b'},
		map[string]string{"a": "b"},
		nil,
	},
	{
		[]byte{0x2, 0x0},
		map[string]string{},
		nil,
	},
	{
		[]byte{0x2, 0x1},
		nil,
		errMalformedPacket,
	},
	{
		[]byte{0x2},
		nil,
		errMalformedPacket,
	},
	{
		[]byte{0x2, 0x8, '/', 'c', '/', 'd', '?', 'a', '=', 'b'},
		map[string]string{"a": "b"},
		nil,
	},
	{
		[]byte{0x2, 0x2, '/', '?', 0x2, 0x3, 'a', '=', 'b'},
		map[string]string{"a": "b"},
		nil,
	},
	{
		[]byte{0x2, 0x9, '/', '?', 'a', '=', 'b', '%', '2', '0', 'c'},
		map[string]string{"a": "b c"},
		nil,
	},
}

func TestHandleOptionalParameters(t *testing.T) {
	for _, tt := range table {
		t.Run(fmt.Sprintf("%#v as %#v", tt.data, tt.values), func(t *testing.T) {
			params, err := handleOptionalParameters(tt.data)
			if err != tt.err {
				if tt.err == nil {
					t.Fatalf("expected no parsing error for %x but got %s", tt.data, err)
				} else {
					t.Fatalf("expected parsing error for %x", tt.data)
				}
			}
			if tt.values != nil {
				if params == nil {
					t.Fatalf("expected values %v for %x", tt.values, tt.data)
				} else {
					for key, want := range tt.values {
						if got, ok := params.String(key); !ok {
							t.Fatalf("params missing entry %s for data %x", key, tt.data)
						} else if got != want {
							t.Fatalf("expected param %s=%s, but was %s for data %x", key, want, got, tt.data)
						}
					}
				}
			}
		})
	}
}
