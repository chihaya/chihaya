// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package tracker implements a protocol-independent, middleware-composed
// BitTorrent tracker.
package tracker

import (
	"errors"

	"fmt"
	"github.com/chihaya/chihaya"
)

// ClientError represents an error that should be exposed to the client over
// the BitTorrent protocol implementation.
type ClientError string

// Error implements the error interface for ClientError.
func (c ClientError) Error() string { return string(c) }

// Tracker represents a protocol-independent, middleware-composed BitTorrent
// tracker.
type Tracker struct {
	cfg            *chihaya.TrackerConfig
	handleAnnounce AnnounceHandler
	handleScrape   ScrapeHandler
}

// NewTracker constructs a newly allocated Tracker composed of the middleware
// in the provided configuration.
func NewTracker(cfg *chihaya.TrackerConfig) (*Tracker, error) {
	var achain announceChain
	for _, mwConfig := range cfg.AnnounceMiddleware {
		mw, ok := announceMiddlewareConstructors[mwConfig.Name]
		if !ok {
			return nil, errors.New("failed to find announce middleware: " + mwConfig.Name)
		}
		middleware, err := mw(mwConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to load announce middleware %q: %s", mwConfig.Name, err.Error())
		}
		achain.Append(middleware)
	}

	var schain scrapeChain
	for _, mwConfig := range cfg.ScrapeMiddleware {
		mw, ok := scrapeMiddlewareConstructors[mwConfig.Name]
		if !ok {
			return nil, errors.New("failed to find scrape middleware: " + mwConfig.Name)
		}
		middleware, err := mw(mwConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to load scrape middleware %q: %s", mwConfig.Name, err.Error())
		}
		schain.Append(middleware)
	}

	return &Tracker{
		cfg:            cfg,
		handleAnnounce: achain.Handler(),
		handleScrape:   schain.Handler(),
	}, nil
}

// HandleAnnounce runs an AnnounceRequest through the Tracker's middleware and
// returns the result.
func (t *Tracker) HandleAnnounce(req *chihaya.AnnounceRequest) (*chihaya.AnnounceResponse, error) {
	resp := &chihaya.AnnounceResponse{}
	err := t.handleAnnounce(t.cfg, req, resp)
	return resp, err
}

// HandleScrape runs a ScrapeRequest through the Tracker's middleware and
// returns the result.
func (t *Tracker) HandleScrape(req *chihaya.ScrapeRequest) (*chihaya.ScrapeResponse, error) {
	resp := &chihaya.ScrapeResponse{}
	err := t.handleScrape(t.cfg, req, resp)
	return resp, err
}
