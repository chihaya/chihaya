// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package query implements a faster single-purpose URL Query parser.
package query

import (
	"errors"
	"net/url"
	"strconv"
	"strings"
)

// Query represents a parsed URL.Query.
type Query struct {
	Infohashes []string
	Params     map[string]string
}

// New parses a raw url query.
func New(query string) (*Query, error) {
	var (
		keyStart, keyEnd int
		valStart, valEnd int
		firstInfohash    string

		onKey       = true
		hasInfohash = false

		q = &Query{
			Infohashes: nil,
			Params:     make(map[string]string),
		}
	)

	for i, length := 0, len(query); i < length; i++ {
		separator := query[i] == '&' || query[i] == ';' || query[i] == '?'
		if separator || i == length-1 {
			if onKey {
				keyStart = i + 1
				continue
			}

			if i == length-1 && !separator {
				if query[i] == '=' {
					continue
				}
				valEnd = i
			}

			keyStr, err := url.QueryUnescape(query[keyStart : keyEnd+1])
			if err != nil {
				return nil, err
			}

			valStr, err := url.QueryUnescape(query[valStart : valEnd+1])
			if err != nil {
				return nil, err
			}

			q.Params[strings.ToLower(keyStr)] = valStr

			if keyStr == "info_hash" {
				if hasInfohash {
					// Multiple infohashes
					if q.Infohashes == nil {
						q.Infohashes = []string{firstInfohash}
					}
					q.Infohashes = append(q.Infohashes, valStr)
				} else {
					firstInfohash = valStr
					hasInfohash = true
				}
			}

			onKey = true
			keyStart = i + 1

		} else if query[i] == '=' {
			onKey = false
			valStart = i + 1
		} else if onKey {
			keyEnd = i
		} else {
			valEnd = i
		}
	}

	return q, nil
}

// Uint64 is a helper to obtain a uint of any length from a Query. After being
// called, you can safely cast the uint64 to your desired length.
func (q *Query) Uint64(key string) (uint64, error) {
	str, exists := q.Params[key]
	if !exists {
		return 0, errors.New("value does not exist for key: " + key)
	}

	val, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		return 0, err
	}

	return val, nil
}
