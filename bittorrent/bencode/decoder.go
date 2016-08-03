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
	"bufio"
	"bytes"
	"errors"
	"io"
	"strconv"
)

// A Decoder reads bencoded objects from an input stream.
type Decoder struct {
	r *bufio.Reader
}

// NewDecoder returns a new decoder that reads from r.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: bufio.NewReader(r)}
}

// Decode unmarshals the next bencoded value in the stream.
func (dec *Decoder) Decode() (interface{}, error) {
	return unmarshal(dec.r)
}

// Unmarshal deserializes and returns the bencoded value in buf.
func Unmarshal(buf []byte) (interface{}, error) {
	r := bufio.NewReader(bytes.NewBuffer(buf))
	return unmarshal(r)
}

// unmarshal reads bencoded values from a bufio.Reader
func unmarshal(r *bufio.Reader) (interface{}, error) {
	tok, err := r.ReadByte()
	if err != nil {
		return nil, err
	}

	switch tok {
	case 'i':
		return readTerminatedInt(r, 'e')

	case 'l':
		list := NewList()
		for {
			ok, err := readTerminator(r, 'e')
			if err != nil {
				return nil, err
			} else if ok {
				break
			}

			v, err := unmarshal(r)
			if err != nil {
				return nil, err
			}
			list = append(list, v)
		}
		return list, nil

	case 'd':
		dict := NewDict()
		for {
			ok, err := readTerminator(r, 'e')
			if err != nil {
				return nil, err
			} else if ok {
				break
			}

			v, err := unmarshal(r)
			if err != nil {
				return nil, err
			}

			key, ok := v.(string)
			if !ok {
				return nil, errors.New("bencode: non-string map key")
			}

			dict[key], err = unmarshal(r)
			if err != nil {
				return nil, err
			}
		}
		return dict, nil

	default:
		err = r.UnreadByte()
		if err != nil {
			return nil, err
		}

		length, err := readTerminatedInt(r, ':')
		if err != nil {
			return nil, errors.New("bencode: unknown input sequence")
		}

		buf := make([]byte, length)
		n, err := r.Read(buf)

		if err != nil {
			return nil, err
		} else if int64(n) != length {
			return nil, errors.New("bencode: short read")
		}

		return string(buf), nil
	}
}

func readTerminator(r io.ByteScanner, term byte) (bool, error) {
	tok, err := r.ReadByte()
	if err != nil {
		return false, err
	} else if tok == term {
		return true, nil
	}
	return false, r.UnreadByte()
}

func readTerminatedInt(r *bufio.Reader, term byte) (int64, error) {
	buf, err := r.ReadSlice(term)
	if err != nil {
		return 0, err
	} else if len(buf) <= 1 {
		return 0, errors.New("bencode: empty integer field")
	}

	return strconv.ParseInt(string(buf[:len(buf)-1]), 10, 64)
}
