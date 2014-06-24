// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package server

import (
	"io"
	"net/http"

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

	if s.conf.Private {
		_, err = conn.FindUser(scrape.Passkey)
		if err != nil {
			fail(err, w, r)
			return
		}
	}

	var torrents []*models.Torrent
	for _, infohash := range scrape.Infohashes {
		torrent, err := conn.FindTorrent(infohash)
		if err != nil {
			fail(err, w, r)
			return
		}
		torrents = append(torrents, torrent)
	}

	bencoder := bencode.NewEncoder(w)
	bencoder.Encode("d")
	bencoder.Encode("files")
	for _, torrent := range torrents {
		writeTorrentStatus(w, torrent)
	}
	bencoder.Encode("e")

	log.V(3).Infof("chihaya: handled scrape from %s", r.RemoteAddr)

	w.(http.Flusher).Flush()
}

func writeTorrentStatus(w io.Writer, t *models.Torrent) {
	bencoder := bencode.NewEncoder(w)
	bencoder.Encode("t.Infohash")
	bencoder.Encode("d")
	bencoder.Encode("complete")
	bencoder.Encode(len(t.Seeders))
	bencoder.Encode("downloaded")
	bencoder.Encode(t.Snatches)
	bencoder.Encode("incomplete")
	bencoder.Encode(len(t.Leechers))
	bencoder.Encode("e")
}
