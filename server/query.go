// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package server

import (
	"errors"
	"net/url"
	"strconv"
)

// parsedQuery represents a parsed URL.Query.
type parsedQuery struct {
	Infohashes []string
	Params     map[string]string
}

func (pq *parsedQuery) getUint64(key string) (uint64, error) {
	str, exists := pq.Params[key]
	if !exists {
		return 0, errors.New("Value does not exist for key: " + key)
	}
	val, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		return 0, err
	}
	return val, nil
}

// parseQuery parses a raw url query.
func parseQuery(query string) (*parsedQuery, error) {
	var (
		keyStart, keyEnd int
		valStart, valEnd int
		firstInfohash    string

		onKey       = true
		hasInfohash = false

		pq = &parsedQuery{
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

			pq.Params[keyStr] = valStr

			if keyStr == "info_hash" {
				if hasInfohash {
					// Multiple infohashes
					if pq.Infohashes == nil {
						pq.Infohashes = []string{firstInfohash}
					}
					pq.Infohashes = append(pq.Infohashes, valStr)
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
	return pq, nil
}
