// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package tracker provides an abstraction of the behavior of a BitTorrent
// tracker that is both storage and transport protocol agnostic.
package tracker

import (
	"time"

	"github.com/golang/glog"

	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/deltastore"
	"github.com/chihaya/chihaya/store"
	"github.com/chihaya/chihaya/tracker/models"
)

// Tracker represents the logic necessary to service BitTorrent announces,
// independently of the underlying data transports used.
type Tracker struct {
	Config     *config.Config
	Store      store.Conn
	DeltaStore deltastore.Conn
}

// Server represents a server for a given BitTorrent tracker protocol.
type Server interface {
	// Serve runs the server and blocks until the server has shut down.
	Serve(addr string)

	// Stop cleanly shuts down the server in a non-blocking manner.
	Stop()
}

// New creates a new Tracker, and opens any necessary connections.
// Maintenance routines are automatically spawned in the background.
func New(cfg *config.Config) (*Tracker, error) {
	storeConn, err := store.Open(&cfg.StoreConfig)
	if err != nil {
		return nil, err
	}

	deltaConn, err := deltastore.Open(&cfg.DeltaStoreConfig)
	if err != nil {
		return nil, err
	}

	tkr := &Tracker{
		Config:     cfg,
		Store:      storeConn,
		DeltaStore: deltaConn,
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
	if err := tkr.DeltaStore.Close(); err != nil {
		return err
	}
	return tkr.Store.Close()
}

// LoadApprovedClients loads a list of client IDs into the tracker's storage.
func (tkr *Tracker) LoadApprovedClients(clients []string) {
	for _, client := range clients {
		err := tkr.Store.PutClient(client)
		if err != nil {
			glog.Errorf("failed to load %s into whitelist: %s", client, err)
		}
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

		err := tkr.Store.PurgeInactivePeers(purgeEmptyTorrents, before)
		if err != nil {
			glog.Errorf("Error purging torrents: %s", err)
		}
	}
}
