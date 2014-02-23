// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package server

import (
	"errors"
	"io"
	"log"
	"net/http"
	"path"

	"github.com/chihaya/chihaya/storage"
)

func (s *Server) serveScrape(w http.ResponseWriter, r *http.Request) {
	// Parse the query
	pq, err := parseQuery(r.URL.RawQuery)
	if err != nil {
		fail(errors.New("Error parsing query"), w, r)
		return
	}

	// Get a connection to the tracker db
	conn, err := s.trackerPool.Get()
	if err != nil {
		log.Fatal(err)
	}

	// Find and validate the user
	passkey, _ := path.Split(r.URL.Path)
	_, err = validateUser(conn, passkey)
	if err != nil {
		fail(err, w, r)
		return
	}

	io.WriteString(w, "d")
	writeBencoded(w, "files")
	if pq.Infohashes != nil {
		for _, infohash := range pq.Infohashes {
			torrent, exists, err := conn.FindTorrent(infohash)
			if err != nil {
				log.Panicf("server: %s", err)
			}
			if exists {
				writeBencoded(w, infohash)
				writeScrapeInfo(w, torrent)
			}
		}
	} else if infohash, exists := pq.Params["info_hash"]; exists {
		torrent, exists, err := conn.FindTorrent(infohash)
		if err != nil {
			log.Panicf("server: %s", err)
		}
		if exists {
			writeBencoded(w, infohash)
			writeScrapeInfo(w, torrent)
		}
	}
	io.WriteString(w, "e")

	w.(http.Flusher).Flush()
}

func writeScrapeInfo(w io.Writer, torrent *storage.Torrent) {
	io.WriteString(w, "d")
	writeBencoded(w, "complete")
	writeBencoded(w, len(torrent.Seeders))
	writeBencoded(w, "downloaded")
	writeBencoded(w, torrent.Snatches)
	writeBencoded(w, "incomplete")
	writeBencoded(w, len(torrent.Leechers))
	io.WriteString(w, "e")
}
