// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package query implements a simple, fast URL parser designed to be used to
// parse parameters sent from BitTorrent clients. The last value of a key wins,
// except for they key "info_hash".
package query

import (
	"errors"
	"net/url"
	"strconv"
	"strings"

	"github.com/chihaya/chihaya"
)

// ErrKeyNotFound is returned when a provided key has no value associated with
// it.
var ErrKeyNotFound = errors.New("query: value for the provided key does not exist")

// ErrInvalidInfohash is returned when parsing a query encounters an infohash
// with invalid length.
var ErrInvalidInfohash = errors.New("query: invalid infohash")

// Query represents a parsed URL.Query.
type Query struct {
	query      string
	params     map[string]string
	infoHashes []chihaya.InfoHash
}

// New parses a raw URL query.
func New(query string) (*Query, error) {
	var (
		keyStart, keyEnd int
		valStart, valEnd int

		onKey = true

		q = &Query{
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
				q.infoHashes = append(q.infoHashes, chihaya.InfoHashFromString(valStr))
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
func (q *Query) String(key string) (string, error) {
	val, exists := q.params[key]
	if !exists {
		return "", ErrKeyNotFound
	}
	return val, nil
}

// Uint64 returns a uint parsed from a query. After being called, it is safe to
// cast the uint64 to your desired length.
func (q *Query) Uint64(key string) (uint64, error) {
	str, exists := q.params[key]
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
func (q *Query) InfoHashes() []chihaya.InfoHash {
	return q.infoHashes
}
