// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package server

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
)

type parsedQuery struct {
	infohashes []string
	params     map[string]string
}

func (pq *parsedQuery) getUint64(key string) (uint64, bool) {
	str, exists := pq.params[key]
	if !exists {
		return 0, false
	}
	val, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		return 0, false
	}
	return val, true
}

func parseQuery(query string) (*parsedQuery, error) {
	var (
		keyStart, keyEnd int
		valStart, valEnd int
		firstInfohash    string

		onKey       = true
		hasInfohash = false

		pq = &parsedQuery{
			infohashes: nil,
			params:     make(map[string]string),
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

			pq.params[keyStr] = valStr

			if keyStr == "info_hash" {
				if hasInfohash {
					// Multiple infohashes
					if pq.infohashes == nil {
						pq.infohashes = []string{firstInfohash}
					}
					pq.infohashes = append(pq.infohashes, valStr)
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

func (pq *parsedQuery) validate() error {
	infohash, ok := pq.params["info_hash"]
	if infohash == "" {
		return errors.New("infohash does not exist")
	}
	peerId, ok := pq.params["peer_id"]
	if peerId == "" {
		return errors.New("peerId does not exist")
	}
	port, ok := pq.getUint64("port")
	if ok == false {
		return errors.New("port does not exist")
	}
	uploaded, ok := pq.getUint64("uploaded")
	if ok == false {
		return errors.New("uploaded does not exist")
	}
	downloaded, ok := pq.getUint64("downloaded")
	if ok == false {
		return errors.New("downloaded does not exist")
	}
	left, ok := pq.getUint64("left")
	if ok == false {
		return errors.New("left does not exist")
	}
	return nil
}

// TODO IPv6 support
func (pq *parsedQuery) determineIP(r *http.Request) (string, error) {
	ip, ok := pq.params["ip"]
	if !ok {
		ip, ok = pq.params["ipv4"]
		if !ok {
			ips, ok := r.Header["X-Real-Ip"]
			if ok && len(ips) > 0 {
				ip = ips[0]
			} else {
				portIndex := len(r.RemoteAddr) - 1
				for ; portIndex >= 0; portIndex-- {
					if r.RemoteAddr[portIndex] == ':' {
						break
					}
				}
				if portIndex != -1 {
					ip = r.RemoteAddr[0:portIndex]
				} else {
					return "", errors.New("Failed to parse IP address")
				}
			}
		}
	}
	return ip, nil
}
