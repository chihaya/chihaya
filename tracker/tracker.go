// Copyright 2014 The Chihaya Authors. All rights reserved.
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
	cfg     *config.Config
	Pool    Pool
	backend backend.Conn
}

// New creates a new Tracker, and opens any necessary connections.
// Maintenance routines are automatically spawned in the background.
func New(cfg *config.Config) (*Tracker, error) {
	pool, err := Open(&cfg.Tracker)
	if err != nil {
		return nil, err
	}

	bc, err := backend.Open(&cfg.Backend)
	if err != nil {
		return nil, err
	}

	go purgeInactivePeers(
		pool,
		cfg.PurgeInactiveTorrents,
		cfg.Announce.Duration*2,
		cfg.Announce.Duration,
	)

	tkr := &Tracker{
		cfg:     cfg,
		Pool:    pool,
		backend: bc,
	}

	if cfg.ClientWhitelistEnabled {
		tkr.LoadApprovedClients(cfg.ClientWhitelist)
	}

	return tkr, nil
}

// Close gracefully shutdowns a Tracker by closing any database connections.
func (tkr *Tracker) Close() (err error) {
	err = tkr.Pool.Close()
	if err != nil {
		return
	}

	err = tkr.backend.Close()
	if err != nil {
		return
	}

	return
}

// LoadApprovedClients loads a list of client IDs into the tracker's storage.
func (tkr *Tracker) LoadApprovedClients(clients []string) error {
	conn, err := tkr.Pool.Get()
	if err != nil {
		return err
	}

	for _, client := range clients {
		err = conn.PutClient(client)
		if err != nil {
			return err
		}
	}

	return nil
}

// Writer serializes a tracker's responses, and is implemented for each
// response transport used by the tracker.
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
func purgeInactivePeers(p Pool, purgeEmptyTorrents bool, threshold, interval time.Duration) {
	for _ = range time.NewTicker(interval).C {
		before := time.Now().Add(-threshold)
		glog.V(0).Infof("Purging peers with no announces since %s", before)

		conn, err := p.Get()

		if err != nil {
			glog.Error("Unable to get connection for a routine")
			continue
		}

		err = conn.PurgeInactivePeers(purgeEmptyTorrents, before)
		if err != nil {
			glog.Errorf("Error purging torrents: %s", err)
		}

		conn.Close()
	}
}
