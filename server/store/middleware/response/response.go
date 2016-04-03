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
	tracker.RegisterAnnounceMiddleware("store_response", responseAnnounceClient)
	tracker.RegisterScrapeMiddleware("store_response", responseScrapeClient)
}

// FailedToRetrievePeers represents an error that has been return when attempting to fetch peers from the store.
type FailedToRetrievePeers string

// Error interface for FailedToRetrievePeers.
func (f FailedToRetrievePeers) Error() string { return string(f) }

// responseAnnounceClient provides a middleware to make a response to an
// announce based on the current request.
func responseAnnounceClient(next tracker.AnnounceHandler) tracker.AnnounceHandler {
	return func(cfg *chihaya.TrackerConfig, req *chihaya.AnnounceRequest, resp *chihaya.AnnounceResponse) (err error) {
		storage := store.MustGetStore()

		resp.MinInterval = cfg.MinAnnounceInterval
		resp.Compact = req.Compact
		resp.Complete = int32(storage.NumSeeders(req.InfoHash))
		resp.Incomplete = int32(storage.NumLeechers(req.InfoHash))
		resp.IPv4Peers, resp.IPv6Peers, err = storage.AnnouncePeers(req.InfoHash, req.Left == 0, int(req.NumWant))
		if err != nil {
			return err.(FailedToRetrievePeers)
		}

		return next(cfg, req, resp)
	}
}

// responseScrapeClient provides a middleware to make a response to an
// scrape based on the current request.
func responseScrapeClient(next tracker.ScrapeHandler) tracker.ScrapeHandler {
	return func(cfg *chihaya.TrackerConfig, req *chihaya.ScrapeRequest, resp *chihaya.ScrapeResponse) (err error) {
		storage := store.MustGetStore()
		for _, infoHash := range req.InfoHashes {
			resp.Files[infoHash] = chihaya.Scrape{
				Complete:   int32(storage.NumSeeders(infoHash)),
				Incomplete: int32(storage.NumLeechers(infoHash)),
			}
		}

		return next(cfg, req, resp)
	}
}
