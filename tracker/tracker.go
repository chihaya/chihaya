// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.package middleware

package tracker

import (
	"errors"

	"github.com/chihaya/chihaya"
	"github.com/chihaya/chihaya/config"
)

type ClientError string

func (c ClientError) Error() string { return string(c) }

// Tracker represents a protocol independent, middleware-composed BitTorrent
// tracker.
type Tracker struct {
	cfg            *config.TrackerConfig
	handleAnnounce AnnounceHandler
	handleScrape   ScrapeHandler
}

// NewTracker parses a config and generates a Tracker composed by the middleware
// specified in the config.
func NewTracker(cfg *config.TrackerConfig) (*Tracker, error) {
	var achain announceChain
	for _, mwName := range cfg.AnnounceMiddleware {
		mw, ok := announceMiddleware[mwName]
		if !ok {
			return nil, errors.New("failed to find announce middleware: " + mwName)
		}
		achain.Append(mw)
	}

	var schain scrapeChain
	for _, mwName := range cfg.ScrapeMiddleware {
		mw, ok := scrapeMiddleware[mwName]
		if !ok {
			return nil, errors.New("failed to find scrape middleware: " + mwName)
		}
		schain.Append(mw)
	}

	return &Tracker{
		cfg:            cfg,
		handleAnnounce: achain.Handler(),
		handleScrape:   schain.Handler(),
	}, nil
}

// HandleAnnounce runs an AnnounceRequest through a Tracker's middleware and
// returns the result.
func (t *Tracker) HandleAnnounce(req *chihaya.AnnounceRequest) (*chihaya.AnnounceResponse, error) {
	resp := &chihaya.AnnounceResponse{}
	err := t.handleAnnounce(t.cfg, req, resp)
	return resp, err
}

// HandleScrape runs a ScrapeRequest through a Tracker's middleware and returns
// the result.
func (t *Tracker) HandleScrape(req *chihaya.ScrapeRequest) (*chihaya.ScrapeResponse, error) {
	resp := &chihaya.ScrapeResponse{}
	err := t.handleScrape(t.cfg, req, resp)
	return resp, err
}
