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
)

func (s *Server) serveAnnounce(w http.ResponseWriter, r *http.Request) {
	passkey, _ := path.Split(r.URL.Path)
	_, err := validatePasskey(passkey, s.storage)
	if err != nil {
		fail(err, w, r)
		return
	}

	pq, err := parseQuery(r.URL.RawQuery)
	if err != nil {
		fail(errors.New("Error parsing query"), w, r)
		return
	}

	_, err = pq.determineIP(r)
	if err != nil {
		fail(err, w, r)
		return
	}

	err = pq.validate()
	if err != nil {
		fail(errors.New("Malformed request"), w, r)
		return
	}

	if !s.conf.Whitelisted(pq.params["peerId"]) {
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
