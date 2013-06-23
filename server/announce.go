// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package server

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"path"

	"github.com/pushrax/chihaya/config"
)

func (s *Server) serveAnnounce(w http.ResponseWriter, r *http.Request) {
	passkey, _ := path.Split(r.URL.Path)
	user, err := validatePasskey(passkey, s.storage)
	if err != nil {
		fail(err, w, r)
		return
	}

	pq, err := parseQuery(r.URL.RawQuery)
	if err != nil {
		fail(errors.New("Error parsing query"), w, r)
		return
	}

	ip, err := pq.determineIP(r)
	if err != nil {
		fail(err, w, r)
		return
	}

	err = validateParsedQuery(pq)
	if err != nil {
		fail(errors.New("Malformed request"), w, r)
		return
	}

	if !whitelisted(pq.params["peerId"], s.conf) {
		fail(errors.New("Your client is not approved"), w, r)
		return
	}

	torrent, exists, err := s.storage.FindTorrent(pq.params["infohash"])
	if err != nil {
		log.Panicf("server: %s", err)
	}
	if !exists {
		fail(errors.New("This torrent does not exist"), w, r)
		return
	}

	if left, _ := pq.getUint64("left"); torrent.Status == 1 && left == 0 {
		err := s.storage.UnpruneTorrent(torrent)
		if err != nil {
			log.Panicf("server: %s", err)
		}
		torrent.Status = 0
	} else if torrent.Status != 0 {
		fail(
			fmt.Errorf(
				"This torrent does not exist (status: %d, left: %d)",
				torrent.Status,
				left,
			),
			w,
			r,
		)
		return
	}

	// TODO continue
}

func whitelisted(peerId string, conf *config.Config) bool {
	var (
		widLen  int
		matched bool
	)

	for _, whitelistedId := range conf.Whitelist {
		widLen = len(whitelistedId)
		if widLen <= len(peerId) {
			matched = true
			for i := 0; i < widLen; i++ {
				if peerId[i] != whitelistedId[i] {
					matched = false
					break
				}
			}
			if matched {
				return true
			}
		}
	}
	return false
}
