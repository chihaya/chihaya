// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.package middleware

package tracker

import (
	"github.com/chihaya/chihaya"
	"github.com/chihaya/chihaya/config"
)

// AnnounceHandler is a function that operates on an AnnounceResponse before it
// has been delivered to a client.
type AnnounceHandler func(*config.TrackerConfig, chihaya.AnnounceRequest, *chihaya.AnnounceResponse) error

func (h AnnounceHandler) handleAnnounce(cfg *config.TrackerConfig, req chihaya.AnnounceRequest, resp *chihaya.AnnounceResponse) error {
	return h(cfg, req, resp)
}

// AnnounceMiddleware is higher-order AnnounceHandler used to implement modular
// behavior processing an announce.
type AnnounceMiddleware func(AnnounceHandler) AnnounceHandler

type announceChain struct{ mw []AnnounceMiddleware }

func (c *announceChain) Append(mw ...AnnounceMiddleware) {
	newMW := make([]AnnounceMiddleware, len(c.mw)+len(mw))
	copy(newMW[:len(c.mw)], c.mw)
	copy(newMW[len(c.mw):], mw)
	c.mw = newMW
}

func (c *announceChain) Handler() AnnounceHandler {
	final := func(cfg *config.TrackerConfig, req chihaya.AnnounceRequest, resp *chihaya.AnnounceResponse) error {
		return nil
	}
	for i := len(c.mw) - 1; i >= 0; i-- {
		final = c.mw[i](final)
	}
	return final
}

var announceMiddleware = make(map[string]AnnounceMiddleware)

// RegisterAnnounceMiddleware makes a middleware available to the tracker under
// the provided named.
//
// If this function is called twice with the same name or if the handler is nil,
// it panics.
func RegisterAnnounceMiddleware(name string, mw AnnounceMiddleware) {
	if mw == nil {
		panic("tracker: could not register nil AnnounceMiddleware")
	}

	if _, dup := announceMiddleware[name]; dup {
		panic("tracker: could not register duplicate AnnounceMiddleware: " + name)
	}

	announceMiddleware[name] = mw
}

// ScrapeHandler is a middleware function that operates on a ScrapeResponse
// before it has been delivered to a client.
type ScrapeHandler func(*config.TrackerConfig, chihaya.ScrapeRequest, *chihaya.ScrapeResponse) error

func (h ScrapeHandler) handleScrape(cfg *config.TrackerConfig, req chihaya.ScrapeRequest, resp *chihaya.ScrapeResponse) error {
	return h(cfg, req, resp)
}

// ScrapeMiddleware is higher-order ScrapeHandler used to implement modular
// behavior processing a scrape.
type ScrapeMiddleware func(ScrapeHandler) ScrapeHandler

type scrapeChain struct{ mw []ScrapeMiddleware }

func (c *scrapeChain) Append(mw ...ScrapeMiddleware) {
	newMW := make([]ScrapeMiddleware, len(c.mw)+len(mw))
	copy(newMW[:len(c.mw)], c.mw)
	copy(newMW[len(c.mw):], mw)
	c.mw = newMW
}

func (c *scrapeChain) Handler() ScrapeHandler {
	final := func(cfg *config.TrackerConfig, req chihaya.ScrapeRequest, resp *chihaya.ScrapeResponse) error {
		return nil
	}
	for i := len(c.mw) - 1; i >= 0; i-- {
		final = c.mw[i](final)
	}
	return final
}

var scrapeMiddleware = make(map[string]ScrapeMiddleware)

// RegisterScrapeMiddleware makes a middleware available to the tracker under
// the provided named.
//
// If this function is called twice with the same name or if the handler is nil,
// it panics.
func RegisterScrapeMiddleware(name string, mw ScrapeMiddleware) {
	if mw == nil {
		panic("tracker: could not register nil ScrapeMiddleware")
	}

	if _, dup := scrapeMiddleware[name]; dup {
		panic("tracker: could not register duplicate ScrapeMiddleware: " + name)
	}

	scrapeMiddleware[name] = mw
}
