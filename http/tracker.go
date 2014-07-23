// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package http

import (
	"net/http"

	"github.com/julienschmidt/httprouter"

	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/http/query"
	"github.com/chihaya/chihaya/tracker/models"
)

// NewAnnounce parses an HTTP request and generates a models.Announce.
func NewAnnounce(cfg *config.Config, r *http.Request, p httprouter.Params) (*models.Announce, error) {
	q, err := query.New(r.URL.RawQuery)
	if err != nil {
		return nil, err
	}

	compact := q.Params["compact"] != "0"
	event, _ := q.Params["event"]
	numWant := q.RequestedPeerCount(cfg.NumWantFallback)

	infohash, exists := q.Params["info_hash"]
	if !exists {
		return nil, models.ErrMalformedRequest
	}

	peerID, exists := q.Params["peer_id"]
	if !exists {
		return nil, models.ErrMalformedRequest
	}

	ip, err := q.RequestedIP(r, cfg.AllowIPSpoofing)
	if err != nil {
		return nil, models.ErrMalformedRequest
	}

	port, err := q.Uint64("port")
	if err != nil {
		return nil, models.ErrMalformedRequest
	}

	left, err := q.Uint64("left")
	if err != nil {
		return nil, models.ErrMalformedRequest
	}

	downloaded, err := q.Uint64("downloaded")
	if err != nil {
		return nil, models.ErrMalformedRequest
	}

	uploaded, err := q.Uint64("uploaded")
	if err != nil {
		return nil, models.ErrMalformedRequest
	}

	return &models.Announce{
		Config:     cfg,
		Compact:    compact,
		Downloaded: downloaded,
		Event:      event,
		IP:         ip,
		Infohash:   infohash,
		Left:       left,
		NumWant:    numWant,
		Passkey:    p.ByName("passkey"),
		PeerID:     peerID,
		Port:       port,
		Uploaded:   uploaded,
	}, nil
}

// NewScrape parses an HTTP request and generates a models.Scrape.
func NewScrape(cfg *config.Config, r *http.Request, p httprouter.Params) (*models.Scrape, error) {
	q, err := query.New(r.URL.RawQuery)
	if err != nil {
		return nil, err
	}

	if q.Infohashes == nil {
		if _, exists := q.Params["info_hash"]; !exists {
			// There aren't any infohashes.
			return nil, models.ErrMalformedRequest
		}
		q.Infohashes = []string{q.Params["info_hash"]}
	}

	return &models.Scrape{
		Config: cfg,

		Passkey:    p.ByName("passkey"),
		Infohashes: q.Infohashes,
	}, nil
}
