// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package tracker provides a generic interface for manipulating a
// BitTorrent tracker's fast-moving data.
package tracker

import (
	"time"

	"github.com/golang/glog"

	"github.com/chihaya/chihaya/backend"
	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/tracker/models"
)

// Tracker represents the logic necessary to service BitTorrent announces,
// independently of the underlying data transports used.
type Tracker struct {
	Config  *config.Config
	Backend backend.Conn
	*Storage
}

// New creates a new Tracker, and opens any necessary connections.
// Maintenance routines are automatically spawned in the background.
func New(cfg *config.Config) (*Tracker, error) {
	bc, err := backend.Open(&cfg.DriverConfig)
	if err != nil {
		return nil, err
	}

	tkr := &Tracker{
		Config:  cfg,
		Backend: bc,
		Storage: NewStorage(cfg),
	}

	go tkr.purgeInactivePeers(
		cfg.PurgeInactiveTorrents,
		cfg.Announce.Duration*2,
		cfg.Announce.Duration,
	)

	if cfg.ClientWhitelistEnabled {
		tkr.LoadApprovedClients(cfg.ClientWhitelist)
	}

	return tkr, nil
}

// Close gracefully shutdowns a Tracker by closing any database connections.
func (tkr *Tracker) Close() error {
	return tkr.Backend.Close()
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
//
// The default threshold is 2x the announce interval, which gives delayed
// peers a chance to stay alive, while ensuring the majority of responses
// contain active peers.
//
// The default interval is equal to the announce interval, since this is a
// relatively expensive operation.
func (tkr *Tracker) purgeInactivePeers(purgeEmptyTorrents bool, threshold, interval time.Duration) {
	for _ = range time.NewTicker(interval).C {
		before := time.Now().Add(-threshold)
		glog.V(0).Infof("Purging peers with no announces since %s", before)

		err := tkr.PurgeInactivePeers(purgeEmptyTorrents, before)
		if err != nil {
			glog.Errorf("Error purging torrents: %s", err)
		}
	}
}
