// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package bencode implements bencoding of objects as defined in BEP 3 using
// type assertion over reflection for performance.
package bencode

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"
)

// An Encoder writes Bencoded objects to an output stream.
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
	buf := &bytes.Buffer{}
	err := marshal(buf, v)
	return buf.Bytes(), err
}

// Marshaler is the interface implemented by objects that can marshal
// themselves.
type Marshaler interface {
	MarshalBencode() ([]byte, error)
}

// Marshal writes types bencoded to an io.Writer
func marshal(w io.Writer, data interface{}) error {
	switch v := data.(type) {
	case Marshaler:
		bencoded, err := v.MarshalBencode()
		if err != nil {
			return err
		}
		_, err = w.Write(bencoded)
		if err != nil {
			return err
		}

	case string:
		fmt.Fprintf(w, "%d:%s", len(v), v)

	case int:
		fmt.Fprintf(w, "i%de", v)

	case uint:
		fmt.Fprintf(w, "i%se", strconv.FormatUint(uint64(v), 10))

	case int64:
		fmt.Fprintf(w, "i%se", strconv.FormatInt(v, 10))

	case uint64:
		fmt.Fprintf(w, "i%se", strconv.FormatUint(v, 10))

	case time.Duration: // Assume seconds
		fmt.Fprintf(w, "i%se", strconv.FormatInt(int64(v/time.Second), 10))

	case map[string]interface{}:
		fmt.Fprintf(w, "d")
		for key, val := range v {
			fmt.Fprintf(w, "%s:%s", strconv.Itoa(len(key)), key)
			err := marshal(w, val)
			if err != nil {
				return err
			}
		}
		fmt.Fprintf(w, "e")

	case []string:
		fmt.Fprintf(w, "l")
		for _, val := range v {
			err := marshal(w, val)
			if err != nil {
				return err
			}
		}
		fmt.Fprintf(w, "e")

	default:
		// Although not currently necessary,
		// should handle []interface{} manually; Go can't do it implicitly
		return errors.New("bencode: attempted to marshal unsupported type")
	}

	return nil
}
