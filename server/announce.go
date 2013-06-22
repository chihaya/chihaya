// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package server

import (
	"errors"
	"fmt"
	"net/http"
	"path"

	"github.com/pushrax/chihaya/config"
)

func (h *handler) serveAnnounce(w http.ResponseWriter, r *http.Request) {
	passkey, action := path.Split(r.URL.Path)
	user, err := validatePasskey(passkey, h.storage)
	if err != nil {
		fail(err, w)
		return
	}

	pq, err := parseQuery(r.URL.RawQuery)
	if err != nil {
		fail(errors.New("Error parsing query"), w)
		return
	}

	ip, err := pq.determineIP(r)
	if err != nil {
		fail(err, w)
		return
	}

	err = validateParsedQuery(pq)
	if err != nil {
		fail(errors.New("Malformed request"), w)
		return
	}

	if !whitelisted(pq.params["peerId"], h.conf) {
		fail(errors.New("Your client is not approved"), w)
		return
	}

	torrent, exists, err := h.storage.FindTorrent(pq.params["infohash"])
	if err != nil {
		panic("server: failed to find torrent")
	}
	if !exists {
		fail(errors.New("This torrent does not exist"), w)
		return
	}

	if left, _ := pq.getUint64("left"); torrent.Status == 1 && left == 0 {
		err := h.storage.UnpruneTorrent(torrent)
		if err != nil {
			panic("server: failed to unprune torrent")
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
		)
		return
	}

	// TODO
}

func whitelisted(peerId string, conf *config.Config) bool {
	// TODO
	return false
}
