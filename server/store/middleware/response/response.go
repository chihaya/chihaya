// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package response

import (
	"github.com/chihaya/chihaya"
	"github.com/chihaya/chihaya/server/store"
	"github.com/chihaya/chihaya/tracker"
)

func init() {
	tracker.RegisterAnnounceMiddleware("response", responseAnnounceClient)
	tracker.RegisterScrapeMiddleware("reponse", responseScrapeClient)
}

// ErrCouldntGetPeers is returned when something goes wrong with getting
// peers from the store.
var ErrCouldntGetPeers = tracker.ClientError("couldn't get peers")

// responseAnnounceClient provides a middleware to make a response to an
// announce based on the current request.
func responseAnnounceClient(next tracker.AnnounceHandler) tracker.AnnounceHandler {
	return func(cfg *chihaya.TrackerConfig, req *chihaya.AnnounceRequest, resp *chihaya.AnnounceResponse) (err error) {
		resp.MinInterval = cfg.MinAnnounceInterval
		storage := store.MustGetStore()
		resp.IPv4Peers, resp.IPv6Peers, err = storage.AnnouncePeers(req.InfoHash, req.Left == 0, req.NumWant)
		if err != nil {
			return ErrCouldntGetPeers
		}
		resp.Compact = req.Compact
		resp.Incomplete = storage.NumLeechers(req.InfoHash)
		resp.Complete = storage.NumSeeders(req.InfoHash)
		next(cfg, req, resp)
	}
}

// responseScrapeClient provides a middleware to make a response to an
// scrape based on the current request.
func responseScrapeClient(next tracker.ScrapeClient) tracker.ScrapeHandler {
	return func(cfg *chihaya.TrackerConfig, req *chihaya.ScrapeRequest, resp *chihaya.ScrapeResponse) (err error) {
		var scrape chihaya.Scrape
		storage := store.MustGetStore()
		for infoHash := range req.InfoHashes {
			resp.Files[infoHash] = chihaya.Scrape{
				Complete:   storage.GetSeeders(infoHash),
				Incomplete: storage.GetLeechers(infoHash),
			}
		}
		next(cfg, req, resp)
	}
}
