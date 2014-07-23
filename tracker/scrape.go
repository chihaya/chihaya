// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package tracker

import "github.com/chihaya/chihaya/tracker/models"

// HandleScrape encapsulates all the logic of handling a BitTorrent client's
// scrape without being coupled to any transport protocol.
func (tkr *Tracker) HandleScrape(scrape *models.Scrape, w Writer) (err error) {
	conn, err := tkr.Pool.Get()
	if err != nil {
		return err
	}

	defer conn.Close()

	if tkr.cfg.Private {
		_, err = conn.FindUser(scrape.Passkey)
		if err == models.ErrUserDNE {
			w.WriteError(err)
			return nil
		} else if err != nil {
			return err
		}
	}

	var torrents []*models.Torrent
	for _, infohash := range scrape.Infohashes {
		torrent, err := conn.FindTorrent(infohash)
		if err == models.ErrTorrentDNE {
			w.WriteError(err)
			return nil
		} else if err != nil {
			return err
		}
		torrents = append(torrents, torrent)
	}

	return w.WriteScrape(&models.ScrapeResponse{torrents})
}
