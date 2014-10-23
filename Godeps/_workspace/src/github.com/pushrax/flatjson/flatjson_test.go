package flatjson_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/pushrax/flatjson"
)

type Child struct {
	C int    `json:"CC"`
	D string `json:"CD"`
}

func TestBasicFlatten(t *testing.T) {
	val := &struct {
		A int
		B string
	}{10, "str"}

	expected := flatjson.Map{
		"A": 10.0, // JSON numbers are all float64.
		"B": "str",
	}

	testFlattening(t, val, expected)
}

func TestEmbeddedFlatten(t *testing.T) {
	val := &struct {
		Child       // Embedded.
		Other Child // Regular child.
		A     int
	}{}

	expected := flatjson.Map{
		"A":        0.0,
		"CC":       0.0,
		"CD":       "",
		"Other.CC": 0.0,
		"Other.CD": "",
	}

	testFlattening(t, val, expected)
}

func TestIndirection(t *testing.T) {
	o2 := &Child{5, "6"}

	val := &struct {
		*Child
		Other1 interface{} `json:"O1"`
		Other2 **Child     `json:"O2"`
		Other3 *Child      `json:",omitempty"`
	}{
		Child:  &Child{1, "2"},
		Other1: &Child{3, "4"},
		Other2: &o2,
	}

	expected := flatjson.Map{
		"CC":    1.0,
		"CD":    "2",
		"O1.CC": 3.0,
		"O1.CD": "4",
		"O2.CC": 5.0,
		"O2.CD": "6",
	}

	testFlattening(t, val, expected)
}

type L3 struct{ A string }
type L2 struct{ L3 }
type L1 struct{ L2 }
type L0 struct{ L1 }

func TestDeepNesting(t *testing.T) {
	val := &L0{}
	val.A = "abc"

	expected := flatjson.Map{"A": "abc"}
	testFlattening(t, val, expected)
}

type TL1 struct {
	L2 `json:"L2"`
}
type TL0 struct {
	TL1 `json:"L1"`
}

func TestDeepTagNesting(t *testing.T) {
	val := &TL0{}
	val.A = "abc"

	expected := flatjson.Map{"L1.L2.A": "abc"}
	testFlattening(t, val, expected)
}

func TestValidInputs(t *testing.T) {
	val := &struct{ A int }{10}
	expected := flatjson.Map{"A": 10.0}

	testFlattening(t, val, expected)
	testFlattening(t, &val, expected)
}

func TestInvalidInputs(t *testing.T) {
	testPanic(t, struct{ A int }{})
	testPanic(t, 123)
	testPanic(t, "abc")
}

func testPanic(t *testing.T, val interface{}) {
	defer func() {
		if recover() == nil {
			t.Errorf("Expected panic for input %#v\n", val)
		}
	}()

	testFlattening(t, val, flatjson.Map{})
}

func testFlattening(t *testing.T, val interface{}, expected flatjson.Map) {
	flat := flatjson.Flatten(val)

	enc, err := json.Marshal(flat)
	if err != nil {
		t.Fatal(err)
	}

	got := flatjson.Map{}
	err = json.Unmarshal(enc, &got)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Unmarshalled to unexpected value:\n     got: %#v\nexpected: %#v\n", got, expected)
	}
}
