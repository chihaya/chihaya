// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package tracker

import "github.com/chihaya/chihaya"

// AnnounceHandler is a function that operates on an AnnounceResponse before it
// has been delivered to a client.
type AnnounceHandler func(*chihaya.TrackerConfig, *chihaya.AnnounceRequest, *chihaya.AnnounceResponse) error

// AnnounceMiddleware is a higher-order function used to implement the chaining
// of AnnounceHandlers.
type AnnounceMiddleware func(AnnounceHandler) AnnounceHandler

type announceChain struct{ mw []AnnounceMiddleware }

func (c *announceChain) Append(mw ...AnnounceMiddleware) {
	c.mw = append(c.mw, mw...)
}

func (c *announceChain) Handler() AnnounceHandler {
	final := func(cfg *chihaya.TrackerConfig, req *chihaya.AnnounceRequest, resp *chihaya.AnnounceResponse) error {
		return nil
	}

	for i := len(c.mw) - 1; i >= 0; i-- {
		final = c.mw[i](final)
	}
	return final
}

var announceMiddleware = make(map[string]AnnounceMiddleware)

// RegisterAnnounceMiddleware makes a middleware globally available under the
// provided named.
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

// ScrapeHandler is a function that operates on a ScrapeResponse before it has
// been delivered to a client.
type ScrapeHandler func(*chihaya.TrackerConfig, *chihaya.ScrapeRequest, *chihaya.ScrapeResponse) error

// ScrapeMiddleware is higher-order function used to implement the chaining of
// ScrapeHandlers.
type ScrapeMiddleware func(ScrapeHandler) ScrapeHandler

type scrapeChain struct{ mw []ScrapeMiddleware }

func (c *scrapeChain) Append(mw ...ScrapeMiddleware) {
	c.mw = append(c.mw, mw...)
}

func (c *scrapeChain) Handler() ScrapeHandler {
	final := func(cfg *chihaya.TrackerConfig, req *chihaya.ScrapeRequest, resp *chihaya.ScrapeResponse) error {
		return nil
	}
	for i := len(c.mw) - 1; i >= 0; i-- {
		final = c.mw[i](final)
	}
	return final
}

var scrapeMiddleware = make(map[string]ScrapeMiddleware)

// RegisterScrapeMiddleware makes a middleware globally available under the
// provided named.
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
