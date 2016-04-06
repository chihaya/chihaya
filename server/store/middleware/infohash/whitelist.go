// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package infohash

import (
	"github.com/chihaya/chihaya"
	"github.com/chihaya/chihaya/server/store"
	"github.com/chihaya/chihaya/tracker"
)

func init() {
	tracker.RegisterAnnounceMiddleware("infohash_whitelist", whitelistAnnounceInfohash)
	tracker.RegisterScrapeMiddlewareConstructor("infohash_whitelist", whitelistScrapeInfohash)
}

// PrefixInfohash is the prefix to be used for infohashes.
const PrefixInfohash = "ih-"

// whitelistAnnounceInfohash provides a middleware that only allows announces
// for infohashes that are not stored in a StringStore
func whitelistAnnounceInfohash(next tracker.AnnounceHandler) tracker.AnnounceHandler {
	return func(cfg *chihaya.TrackerConfig, req *chihaya.AnnounceRequest, resp *chihaya.AnnounceResponse) (err error) {
		whitelisted, err := store.MustGetStore().HasString(PrefixInfohash + string(req.InfoHash))

		if err != nil {
			return err
		} else if !whitelisted {
			return ErrBlockedInfohash
		}
		return next(cfg, req, resp)
	}
}

// whitelistScrapeInfohash provides a middleware constructor for a middleware
// that blocks or filters scrape requests based on the infohashes scraped.
//
// The middleware works in two modes: block and filter.
// The block mode blocks a scrape completely if any of the infohashes is
// disallowed.
// The filter mode filters any disallowed infohashes from the scrape,
// potentially leaving an empty scrape.
//
// ErrUnknownMode is returned if the Mode specified in the config is unknown.
func whitelistScrapeInfohash(c chihaya.MiddlewareConfig) (tracker.ScrapeMiddleware, error) {
	cfg, err := newConfig(c)
	if err != nil {
		return nil, err
	}

	switch cfg.Mode {
	case ModeFilter:
		return whitelistFilterScrape, nil
	case ModeBlock:
		return whitelistBlockScrape, nil
	default:
		panic("unknown mode")
	}
}

func whitelistFilterScrape(next tracker.ScrapeHandler) tracker.ScrapeHandler {
	return func(cfg *chihaya.TrackerConfig, req *chihaya.ScrapeRequest, resp *chihaya.ScrapeResponse) (err error) {
		whitelisted := false
		storage := store.MustGetStore()
		infohashes := req.InfoHashes

		for i, ih := range infohashes {
			whitelisted, err = storage.HasString(PrefixInfohash + string(ih))

			if err != nil {
				return err
			} else if !whitelisted {
				req.InfoHashes[i] = req.InfoHashes[len(req.InfoHashes)-1]
				req.InfoHashes = req.InfoHashes[:len(req.InfoHashes)-1]
			}
		}

		return next(cfg, req, resp)
	}
}

func whitelistBlockScrape(next tracker.ScrapeHandler) tracker.ScrapeHandler {
	return func(cfg *chihaya.TrackerConfig, req *chihaya.ScrapeRequest, resp *chihaya.ScrapeResponse) (err error) {
		whitelisted := false
		storage := store.MustGetStore()

		for _, ih := range req.InfoHashes {
			whitelisted, err = storage.HasString(PrefixInfohash + string(ih))

			if err != nil {
				return err
			} else if !whitelisted {
				return ErrBlockedInfohash
			}
		}

		return next(cfg, req, resp)
	}
}
