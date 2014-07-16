// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package http

import (
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

	conn, err := t.pool.Get()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	if t.cfg.Private {
		_, err = conn.(tracker.PrivateConn).FindUser(scrape.Passkey)
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

	resp := bencode.NewDict()
	resp["files"] = filesDict(torrents)

	bencoder := bencode.NewEncoder(w)
	err = bencoder.Encode(resp)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func filesDict(torrents []*models.Torrent) bencode.Dict {
	d := bencode.NewDict()

	for _, torrent := range torrents {
		d[torrent.Infohash] = torrentDict(torrent)
	}

	return d
}

func torrentDict(torrent *models.Torrent) bencode.Dict {
	d := bencode.NewDict()

	d["complete"] = len(torrent.Seeders)
	d["incomplete"] = len(torrent.Leechers)
	d["downloaded"] = torrent.Snatches

	return d
}
