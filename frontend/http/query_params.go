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

package http

import (
	"errors"
	"net/url"
	"strconv"
	"strings"

	"github.com/jzelinskie/trakr/bittorrent"
)

// ErrKeyNotFound is returned when a provided key has no value associated with
// it.
var ErrKeyNotFound = errors.New("http: value for the provided key does not exist")

// ErrInvalidInfohash is returned when parsing a query encounters an infohash
// with invalid length.
var ErrInvalidInfohash = errors.New("http: invalid infohash")

// QueryParams parses an HTTP Query and implements the bittorrent.Params
// interface with some additional helpers.
type QueryParams struct {
	query      string
	params     map[string]string
	infoHashes []bittorrent.InfoHash
}

// NewQueryParams parses a raw URL query.
func NewQueryParams(query string) (*QueryParams, error) {
	var (
		keyStart, keyEnd int
		valStart, valEnd int

		onKey = true

		q = &QueryParams{
			query:      query,
			infoHashes: nil,
			params:     make(map[string]string),
		}
	)

	for i, length := 0, len(query); i < length; i++ {
		separator := query[i] == '&' || query[i] == ';' || query[i] == '?'
		last := i == length-1

		if separator || last {
			if onKey && !last {
				keyStart = i + 1
				continue
			}

			if last && !separator && !onKey {
				valEnd = i
			}

			keyStr, err := url.QueryUnescape(query[keyStart : keyEnd+1])
			if err != nil {
				return nil, err
			}

			var valStr string

			if valEnd > 0 {
				valStr, err = url.QueryUnescape(query[valStart : valEnd+1])
				if err != nil {
					return nil, err
				}
			}

			if keyStr == "info_hash" {
				if len(valStr) != 20 {
					return nil, ErrInvalidInfohash
				}
				q.infoHashes = append(q.infoHashes, bittorrent.InfoHashFromString(valStr))
			} else {
				q.params[strings.ToLower(keyStr)] = valStr
			}

			valEnd = 0
			onKey = true
			keyStart = i + 1

		} else if query[i] == '=' {
			onKey = false
			valStart = i + 1
			valEnd = 0
		} else if onKey {
			keyEnd = i
		} else {
			valEnd = i
		}
	}

	return q, nil
}

// String returns a string parsed from a query. Every key can be returned as a
// string because they are encoded in the URL as strings.
func (qp *QueryParams) String(key string) (string, bool) {
	value, ok := qp.params[key]
	return value, ok
}

// Uint64 returns a uint parsed from a query. After being called, it is safe to
// cast the uint64 to your desired length.
func (qp *QueryParams) Uint64(key string) (uint64, error) {
	str, exists := qp.params[key]
	if !exists {
		return 0, ErrKeyNotFound
	}

	val, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		return 0, err
	}

	return val, nil
}

// InfoHashes returns a list of requested infohashes.
func (qp *QueryParams) InfoHashes() []bittorrent.InfoHash {
	return qp.infoHashes
}
