// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package tracker provides a generic interface for manipulating a
// BitTorrent tracker's fast-moving data.
package tracker

import (
	"time"

	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/tracker/models"
	
	"github.com/mrd0ll4r/logger"
)

// Tracker represents the logic necessary to service BitTorrent announces,
// independently of the underlying data transports used.
type Tracker struct {
	Config *config.Config
	*Storage
}

// New creates a new Tracker, and opens any necessary connections.
// Maintenance routines are automatically spawned in the background.
func New(cfg *config.Config) (*Tracker, error) {
	tkr := &Tracker{
		Config:  cfg,
		Storage: NewStorage(cfg),
	}

	go tkr.purgeInactivePeers(
		cfg.PurgeInactiveTorrents,
		time.Duration(float64(cfg.MinAnnounce.Duration)*cfg.ReapRatio),
		cfg.ReapInterval.Duration,
	)

	if cfg.ClientWhitelistEnabled {
		tkr.LoadApprovedClients(cfg.ClientWhitelist)
	}

	return tkr, nil
}

// Close gracefully shutdowns a Tracker by closing any database connections.
func (tkr *Tracker) Close() error {

	// TODO(jzelinskie): shutdown purgeInactivePeers goroutine.

	return nil
}

// LoadApprovedClients loads a list of client IDs into the tracker's storage.
func (tkr *Tracker) LoadApprovedClients(clients []string) {
	for _, client := range clients {
		tkr.PutClient(client)
	}
}

// Writer serializes a tracker's responses, and is implemented for each
// response transport used by the tracker. Only one of these may be called
// per request, and only once.
//
// Note, data passed into any of these functions will not contain sensitive
// information, so it may be passed back the client freely.
type Writer interface {
	WriteError(err error) error
	WriteAnnounce(*models.AnnounceResponse) error
	WriteScrape(*models.ScrapeResponse) error
}

// purgeInactivePeers periodically walks the torrent database and removes
// peers that haven't announced recently.
func (tkr *Tracker) purgeInactivePeers(purgeEmptyTorrents bool, threshold, interval time.Duration) {
	for _ = range time.NewTicker(interval).C {
		before := time.Now().Add(-threshold)
		logger.Infof("Purging peers with no announces since %s", before)

		err := tkr.PurgeInactivePeers(purgeEmptyTorrents, before)
		if err != nil {
			logger.Warnf("Error purging torrents: %s", err)
		}
	}
}
