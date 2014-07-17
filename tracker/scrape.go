// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package tracker

import (
	"github.com/chihaya/chihaya/tracker/models"
)

func (t *Tracker) HandleScrape(scrape *models.Scrape, w Writer) error {
	conn, err := t.Pool.Get()
	if err != nil {
		return err
	}

	if t.cfg.Private {
		_, err = conn.FindUser(scrape.Passkey)
		if err == ErrUserDNE {
			w.WriteError(err)
			return nil
		} else if err != nil {
			return err
		}
	}

	var torrents []*models.Torrent
	for _, infohash := range scrape.Infohashes {
		torrent, err := conn.FindTorrent(infohash)
		if err == ErrTorrentDNE {
			w.WriteError(err)
			return nil
		} else if err != nil {
			return err
		}
		torrents = append(torrents, torrent)
	}

	return w.WriteScrape(&ScrapeResponse{torrents})
}
