// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package http

import (
	"fmt"
	"io"
	"net/http"

	"github.com/julienschmidt/httprouter"

	"github.com/chihaya/bencode"
	"github.com/chihaya/chihaya/drivers/tracker"
	"github.com/chihaya/chihaya/models"
)

func (t *Tracker) ServeScrape(w http.ResponseWriter, r *http.Request, p httprouter.Params) (int, error) {
	scrape, err := models.NewScrape(t.cfg, r, p)
	if err == models.ErrMalformedRequest {
		fail(w, r, err)
		return http.StatusOK, nil
	} else if err != nil {
		return http.StatusInternalServerError, err
	}

	conn, err := t.tp.Get()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	if t.cfg.Private {
		_, err = conn.FindUser(scrape.Passkey)
		if err == tracker.ErrUserDNE {
			fail(w, r, err)
			return http.StatusOK, nil
		} else if err != nil {
			return http.StatusInternalServerError, err
		}
	}

	var torrents []*models.Torrent
	for _, infohash := range scrape.Infohashes {
		torrent, err := conn.FindTorrent(infohash)
		if err == tracker.ErrTorrentDNE {
			fail(w, r, err)
			return http.StatusOK, nil
		} else if err != nil {
			return http.StatusInternalServerError, err
		}
		torrents = append(torrents, torrent)
	}

	bencoder := bencode.NewEncoder(w)
	fmt.Fprintf(w, "d")
	bencoder.Encode("files")
	for _, torrent := range torrents {
		writeTorrentStatus(w, torrent)
	}
	fmt.Fprintf(w, "e")

	return http.StatusOK, nil
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
