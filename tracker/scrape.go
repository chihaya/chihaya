// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package tracker

import (
	"github.com/chihaya/chihaya/stats"
	"github.com/chihaya/chihaya/tracker/models"
)

// HandleScrape encapsulates all the logic of handling a BitTorrent client's
// scrape without being coupled to any transport protocol.
func (tkr *Tracker) HandleScrape(scrape *models.Scrape, w Writer) (err error) {
	if tkr.Config.PrivateEnabled {
		if _, err = tkr.Store.FindUser(scrape.Passkey); err != nil {
			return err
		}
	}

	var torrents []*models.Torrent
	for _, infohash := range scrape.Infohashes {
		torrent, err := tkr.Store.FindTorrent(infohash)
		if err != nil {
			return err
		}
		torrents = append(torrents, torrent)
	}

	stats.RecordEvent(stats.Scrape)
	return w.WriteScrape(&models.ScrapeResponse{
		Files: torrents,
	})
}
