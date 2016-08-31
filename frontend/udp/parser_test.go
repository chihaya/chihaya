package udp

import "testing"

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
	for _, testCase := range table {
		params, err := handleOptionalParameters(testCase.data)
		if err != testCase.err {
			if testCase.err == nil {
				t.Fatalf("expected no parsing error for %x but got %s", testCase.data, err)
			} else {
				t.Fatalf("expected parsing error for %x", testCase.data)
			}
		}
		if testCase.values != nil {
			if params == nil {
				t.Fatalf("expected values %v for %x", testCase.values, testCase.data)
			} else {
				for key, want := range testCase.values {
					if got, ok := params.String(key); !ok {
						t.Fatalf("params missing entry %s for data %x", key, testCase.data)
					} else if got != want {
						t.Fatalf("expected param %s=%s, but was %s for data %x", key, want, got, testCase.data)
					}
				}
			}
		}
	}
}
