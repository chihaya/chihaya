// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package tracker provides a generic interface for manipulating a
// BitTorrent tracker's fast-moving data.
package tracker

import (
	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/drivers/backend"
	"github.com/chihaya/chihaya/tracker/models"
)

type Tracker struct {
	cfg     *config.Config
	Pool    Pool
	backend backend.Conn
}

func New(cfg *config.Config) (*Tracker, error) {
	pool, err := Open(&cfg.Tracker)
	if err != nil {
		return nil, err
	}

	bc, err := backend.Open(&cfg.Backend)
	if err != nil {
		return nil, err
	}

	go PurgeInactivePeers(
		pool,
		cfg.PurgeInactiveTorrents,
		cfg.Announce.Duration*2,
		cfg.Announce.Duration,
	)

	return &Tracker{
		cfg:     cfg,
		Pool:    pool,
		backend: bc,
	}, nil
}

type Writer interface {
	WriteError(error) error
	WriteAnnounce(*models.AnnounceResponse) error
	WriteScrape(*models.ScrapeResponse) error
}
