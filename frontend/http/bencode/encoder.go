package bencode

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strconv"
	"time"
)

// An Encoder writes bencoded objects to an output stream.
type Encoder struct {
	w io.Writer
}

// NewEncoder returns a new encoder that writes to w.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

// Encode writes the bencoding of v to the stream.
func (enc *Encoder) Encode(v interface{}) error {
	return marshal(enc.w, v)
}

// Marshal returns the bencoding of v.
func Marshal(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	err := marshal(&buf, v)
	return buf.Bytes(), err
}

// Marshaler is the interface implemented by objects that can marshal
// themselves.
type Marshaler interface {
	MarshalBencode() ([]byte, error)
}

// marshal writes types bencoded to an io.Writer.
func marshal(w io.Writer, data interface{}) (err error) {
	switch v := data.(type) {
	case Marshaler:
		var bencoded []byte
		bencoded, err = v.MarshalBencode()
		if err != nil {
			return err
		}
		_, err = w.Write(bencoded)

	case []byte:
		err = marshalBytes(w, v)

	case string:
		err = marshalString(w, v)

	case []string:
		err = marshalStringSlice(w, v)

	case int:
		err = marshalInt(w, int64(v))

	case int16:
		err = marshalInt(w, int64(v))

	case int32:
		err = marshalInt(w, int64(v))

	case int64:
		err = marshalInt(w, int64(v))

	case uint:
		err = marshalUint(w, uint64(v))

	case uint16:
		err = marshalUint(w, uint64(v))

	case uint32:
		err = marshalUint(w, uint64(v))

	case uint64:
		err = marshalUint(w, uint64(v))

	case time.Duration: // Assume seconds
		err = marshalInt(w, int64(v/time.Second))

	case map[string]interface{}:
		err = marshalMap(w, v)

	case []interface{}:
		err = marshalList(w, v)

	case []Dict:
		var interfaceSlice = make([]interface{}, len(v))
		for i, d := range v {
			interfaceSlice[i] = d
		}
		err = marshalList(w, interfaceSlice)

	default:
		return fmt.Errorf("attempted to marshal unsupported type:\n%T", v)
	}

	return err
}

func marshalInt(w io.Writer, v int64) error {
	if _, err := w.Write([]byte{'i'}); err != nil {
		return err
	}

	if _, err := w.Write([]byte(strconv.FormatInt(v, 10))); err != nil {
		return err
	}

	_, err := w.Write([]byte{'e'})
	return err
}

func marshalUint(w io.Writer, v uint64) error {
	if _, err := w.Write([]byte{'i'}); err != nil {
		return err
	}

	if _, err := w.Write([]byte(strconv.FormatUint(v, 10))); err != nil {
		return err
	}

	_, err := w.Write([]byte{'e'})
	return err
}

func marshalBytes(w io.Writer, v []byte) error {
	if _, err := w.Write([]byte(strconv.Itoa(len(v)))); err != nil {
		return err
	}

	if _, err := w.Write([]byte{':'}); err != nil {
		return err
	}

	_, err := w.Write(v)
	return err
}

func marshalString(w io.Writer, v string) error {
	return marshalBytes(w, []byte(v))
}

func marshalStringSlice(w io.Writer, v []string) error {
	if _, err := w.Write([]byte{'l'}); err != nil {
		return err
	}

	for _, val := range v {
		if err := marshal(w, val); err != nil {
			return err
		}
	}

	_, err := w.Write([]byte{'e'})
	return err
}

func marshalList(w io.Writer, v []interface{}) error {
	if _, err := w.Write([]byte{'l'}); err != nil {
		return err
	}

	for _, val := range v {
		if err := marshal(w, val); err != nil {
			return err
		}
	}

	_, err := w.Write([]byte{'e'})
	return err
}

func marshalMap(w io.Writer, v map[string]interface{}) error {
	if _, err := w.Write([]byte{'d'}); err != nil {
		return err
	}

	var keys []string
	for key, _ := range v {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		val := v[key]
		if err := marshalString(w, key); err != nil {
			return err
		}

		if err := marshal(w, val); err != nil {
			return err
		}
	}

	_, err := w.Write([]byte{'e'})
	return err
}
