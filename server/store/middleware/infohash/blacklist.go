// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package infohash

import (
	"net/http"

	"github.com/chihaya/chihaya"
	"github.com/chihaya/chihaya/server/store"
	"github.com/chihaya/chihaya/tracker"
)

func init() {
	tracker.RegisterAnnounceMiddleware("infohash_blacklist", blacklistAnnounceInfohash)
	tracker.RegisterScrapeMiddlewareConstructor("infohash_blacklist", blacklistScrapeInfohash)
	mustGetStore = func() store.StringStore {
		return store.MustGetStore().StringStore
	}

	store.RegisterHandler(http.MethodPut, pathInfohash, handlePutInfohash)
	store.RegisterHandler(http.MethodDelete, pathInfohash, handleDeleteInfohash)
	store.RegisterHandler(http.MethodGet, pathInfohash, handleGetInfohash)
}

// ErrBlockedInfohash is returned by a middleware if any of the infohashes
// contained in an announce or scrape are disallowed.
var ErrBlockedInfohash = tracker.ClientError("disallowed infohash")

var mustGetStore func() store.StringStore

// blacklistAnnounceInfohash provides a middleware that only allows announces
// for infohashes that are not stored in a StringStore.
func blacklistAnnounceInfohash(next tracker.AnnounceHandler) tracker.AnnounceHandler {
	routesActivated.Do(activateRoutes)

	return func(cfg *chihaya.TrackerConfig, req *chihaya.AnnounceRequest, resp *chihaya.AnnounceResponse) (err error) {
		blacklisted, err := mustGetStore().HasString(PrefixInfohash + string(req.InfoHash[:]))
		if err != nil {
			return err
		} else if blacklisted {
			return ErrBlockedInfohash
		}

		return next(cfg, req, resp)
	}
}

// blacklistScrapeInfohash provides a middleware constructor for a middleware
// that blocks or filters scrape requests based on the infohashes scraped.
//
// The middleware works in two modes: block and filter.
// The block mode blocks a scrape completely if any of the infohashes is
// disallowed.
// The filter mode filters any disallowed infohashes from the scrape,
// potentially leaving an empty scrape.
//
// ErrUnknownMode is returned if the Mode specified in the config is unknown.
func blacklistScrapeInfohash(c chihaya.MiddlewareConfig) (tracker.ScrapeMiddleware, error) {
	routesActivated.Do(activateRoutes)

	cfg, err := newConfig(c)
	if err != nil {
		return nil, err
	}

	switch cfg.Mode {
	case ModeFilter:
		return blacklistFilterScrape, nil
	case ModeBlock:
		return blacklistBlockScrape, nil
	default:
		panic("unknown mode")
	}
}

func blacklistFilterScrape(next tracker.ScrapeHandler) tracker.ScrapeHandler {
	return func(cfg *chihaya.TrackerConfig, req *chihaya.ScrapeRequest, resp *chihaya.ScrapeResponse) (err error) {
		blacklisted := false
		storage := mustGetStore()
		infohashes := req.InfoHashes

		for i, ih := range infohashes {
			blacklisted, err = storage.HasString(PrefixInfohash + string(ih[:]))

			if err != nil {
				return err
			} else if blacklisted {
				req.InfoHashes[i] = req.InfoHashes[len(req.InfoHashes)-1]
				req.InfoHashes = req.InfoHashes[:len(req.InfoHashes)-1]
			}
		}

		return next(cfg, req, resp)
	}
}

func blacklistBlockScrape(next tracker.ScrapeHandler) tracker.ScrapeHandler {
	return func(cfg *chihaya.TrackerConfig, req *chihaya.ScrapeRequest, resp *chihaya.ScrapeResponse) (err error) {
		blacklisted := false
		storage := mustGetStore()

		for _, ih := range req.InfoHashes {
			blacklisted, err = storage.HasString(PrefixInfohash + string(ih[:]))

			if err != nil {
				return err
			} else if blacklisted {
				return ErrBlockedInfohash
			}
		}

		return next(cfg, req, resp)
	}
}
