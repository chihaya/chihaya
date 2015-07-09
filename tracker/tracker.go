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
	"github.com/chihaya/chihaya/event/consumer"
	"github.com/chihaya/chihaya/event/producer"
	"github.com/chihaya/chihaya/store"
	"github.com/chihaya/chihaya/tracker/models"
)

// Tracker represents the logic necessary to service BitTorrent announces,
// independently of the underlying data transports used.
type Tracker struct {
	Config   *config.Config
	Store    store.Conn
	Consumer consumer.Consumer
	Producer producer.Producer
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
	store, err := store.Open(&cfg.StoreConfig)
	if err != nil {
		return nil, err
	}

	consumer, err := consumer.Open(&cfg.ConsumerConfig)
	if err != nil {
		return nil, err
	}

	producer, err := producer.Open(&cfg.ProducerConfig)
	if err != nil {
		return nil, err
	}

	tkr := &Tracker{
		Config:   cfg,
		Store:    store,
		Consumer: consumer,
		Producer: producer,
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
	if err := tkr.Consumer.Close(); err != nil {
		return err
	}

	if err := tkr.Producer.Close(); err != nil {
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
