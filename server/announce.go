package server

import (
	"bytes"
	"errors"
	"log"

	"github.com/jzelinskie/chihaya/config"
	"github.com/jzelinskie/chihaya/storage"
)

func (h *handler) serveAnnounce(w *http.ResponseWriter, r *http.Request) {
	buf := h.bufferpool.Take()
	defer h.bufferpool.Give(buf)
	defer h.writeResponse(&w, r, buf)

	user, err := validatePasskey(dir, h.storage)
	if err != nil {
		fail(err, buf)
		return
	}

	pq, err := parseQuery(r.URL.RawQuery)
	if err != nil {
		fail(errors.New("Error parsing query"), buf)
		return
	}

	ip, err := determineIP(r, pq)
	if err != nil {
		fail(err, buf)
		return
	}

	err := validateParsedQuery(pq)
	if err != nil {
		fail(errors.New("Malformed request"), buf)
		return
	}

	if !whitelisted(peerId, h.conf) {
		fail(errors.New("Your client is not approved"), buf)
		return
	}

	torrent, exists, err := h.storage.FindTorrent(infohash)
	if err != nil {
		panic("server: failed to find torrent")
	}
	if !exists {
		fail(errors.New("This torrent does not exist"), buf)
		return
	}

	if torrent.Status == 1 && left == 0 {
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
			buf,
		)
		return
	}

	//go
}

func whitelisted(peerId string, conf config.Config) bool {
	// TODO Decide if whitelist should be in storage or config
}

func newPeer() {
}
