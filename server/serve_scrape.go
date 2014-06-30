// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package server

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	log "github.com/golang/glog"

	"github.com/chihaya/chihaya/bencode"
	"github.com/chihaya/chihaya/models"
)

func (s *Server) serveScrape(w http.ResponseWriter, r *http.Request) {
	scrape, err := models.NewScrape(r, s.conf)
	if err != nil {
		fail(err, w, r)
		return
	}

	conn, err := s.trackerPool.Get()
	if err != nil {
		fail(err, w, r)
	}

	var user *models.User
	if s.conf.Private {
		user, err = conn.FindUser(scrape.Passkey)
		if err != nil {
			fail(err, w, r)
			return
		}
	}

	var (
		torrents   []*models.Torrent
		torrentIDs []string
	)
	for _, infohash := range scrape.Infohashes {
		torrent, err := conn.FindTorrent(infohash)
		if err != nil {
			fail(err, w, r)
			return
		}
		torrents = append(torrents, torrent)
		torrentIDs = append(torrentIDs, string(torrent.ID))
	}

	bencoder := bencode.NewEncoder(w)
	bencoder.Encode("d")
	bencoder.Encode("files")
	for _, torrent := range torrents {
		writeTorrentStatus(w, torrent)
	}
	bencoder.Encode("e")

	w.(http.Flusher).Flush()

	log.V(5).Infof(
		"scrape: ip: %s user: %s torrents: %s",
		r.RemoteAddr,
		user.ID,
		strings.Join(torrentIDs, ", "),
	)
}

func writeTorrentStatus(w io.Writer, t *models.Torrent) {
	bencoder := bencode.NewEncoder(w)
	bencoder.Encode(t.Infohash)
	fmt.Fprintf(w, "d")
	bencoder.Encode("complete")
	bencoder.Encode(len(t.Seeders))
	bencoder.Encode("downloaded")
	bencoder.Encode(t.Snatches)
	bencoder.Encode("incomplete")
	bencoder.Encode(len(t.Leechers))
	fmt.Fprintf(w, "e")
}
