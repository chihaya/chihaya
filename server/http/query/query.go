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
	"github.com/chihaya/chihaya/pkg/event"
)

// ErrKeyNotFound is returned when a provided key has no value associated with
// it.
var ErrKeyNotFound = errors.New("query: value for the provided key does not exist")

// Query represents a parsed URL.Query.
type Query struct {
	query      string
	infohashes []string
	params     map[string]string
}

// New parses a raw URL query.
func New(query string) (*Query, error) {
	var (
		keyStart, keyEnd int
		valStart, valEnd int
		firstInfohash    string

		onKey       = true
		hasInfohash = false

		q = &Query{
			query:      query,
			infohashes: nil,
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

			q.params[strings.ToLower(keyStr)] = valStr

			if keyStr == "info_hash" {
				if hasInfohash {
					// Multiple infohashes
					if q.infohashes == nil {
						q.infohashes = []string{firstInfohash}
					}
					q.infohashes = append(q.infohashes, valStr)
				} else {
					firstInfohash = valStr
					hasInfohash = true
				}
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

// Infohashes returns a list of requested infohashes.
func (q *Query) Infohashes() ([]string, error) {
	if q.infohashes == nil {
		infohash, err := q.String("info_hash")
		if err != nil {
			return nil, err
		}
		return []string{infohash}, nil
	}
	return q.infohashes, nil
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

// AnnounceRequest generates an chihaya.AnnounceRequest with the parameters
// provided by a query.
func (q *Query) AnnounceRequest() (chihaya.AnnounceRequest, error) {
	request := make(chihaya.AnnounceRequest)

	request["query"] = q.query

	eventStr, err := q.String("event")
	if err != nil {
		return nil, errors.New("failed to parse parameter: event")
	}
	request["event"], err = event.New(eventStr)
	if err != nil {
		return nil, errors.New("failed to provide valid client event")
	}

	compactStr, err := q.String("compact")
	if err != nil {
		return nil, errors.New("failed to parse parameter: compact")
	}
	request["compact"] = compactStr != "0"

	request["info_hash"], err = q.String("info_hash")
	if err != nil {
		return nil, errors.New("failed to parse parameter: info_hash")
	}

	request["peer_id"], err = q.String("peer_id")
	if err != nil {
		return nil, errors.New("failed to parse parameter: peer_id")
	}

	request["left"], err = q.Uint64("left")
	if err != nil {
		return nil, errors.New("failed to parse parameter: left")
	}

	request["downloaded"], err = q.Uint64("downloaded")
	if err != nil {
		return nil, errors.New("failed to parse parameter: downloaded")
	}

	request["uploaded"], err = q.Uint64("uploaded")
	if err != nil {
		return nil, errors.New("failed to parse parameter: uploaded")
	}

	request["numwant"], err = q.String("numwant")
	if err != nil {
		return nil, errors.New("failed to parse parameter: numwant")
	}

	request["ip"], err = q.Uint64("port")
	if err != nil {
		return nil, errors.New("failed to parse parameter: port")
	}

	request["port"], err = q.Uint64("port")
	if err != nil {
		return nil, errors.New("failed to parse parameter: port")
	}

	request["ip"], err = q.String("ip")
	if err != nil {
		return nil, errors.New("failed to parse parameter: ip")
	}

	request["ipv4"], err = q.String("ipv4")
	if err != nil {
		return nil, errors.New("failed to parse parameter: ipv4")
	}

	request["ipv6"], err = q.String("ipv6")
	if err != nil {
		return nil, errors.New("failed to parse parameter: ipv6")
	}

	return request, nil
}

// ScrapeRequest generates an chihaya.ScrapeRequeset with the parameters
// provided by a query.
func (q *Query) ScrapeRequest() (chihaya.ScrapeRequest, error) {
	request := make(chihaya.ScrapeRequest)

	var err error
	request["info_hash"], err = q.Infohashes()
	if err != nil {
		return nil, errors.New("failed to parse parameter: info_hash")
	}

	return request, nil
}
