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
	"strconv"
)

func (s *Server) serveAnnounce(w http.ResponseWriter, r *http.Request) {
	conn := s.connPool.Get()
	defer conn.Close()

	passkey, _ := path.Split(r.URL.Path)
	_, err := validatePasskey(passkey, conn)
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

	err = pq.validateAnnounceParams()
	if err != nil {
		fail(errors.New("Malformed request"), w, r)
		return
	}

	if !s.conf.ClientWhitelisted(pq.params["peer_id"]) {
		fail(errors.New("Your client is not approved"), w, r)
		return
	}

	torrent, exists, err := conn.FindTorrent(pq.params["infohash"])
	if err != nil {
		log.Panicf("server: %s", err)
	}
	if !exists {
		fail(errors.New("This torrent does not exist"), w, r)
		return
	}

	tx, err := conn.NewTx()
	if err != nil {
		log.Panicf("server: %s", err)
	}

	if left, _ := pq.getUint64("left"); torrent.Status == 1 && left == 0 {
		err := tx.UnpruneTorrent(torrent)
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

	var numWant int
	if numWantStr, exists := pq.params["numWant"]; exists {
		numWant, err := strconv.Atoi(numWantStr)
		if err != nil {
			numWant = s.conf.DefaultNumWant
		}
	} else {
		numWant = s.conf.DefaultNumWant
	}

	// TODO continue
}
