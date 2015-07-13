// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package nop implements Producer as a no-op.
package nop

import (
	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/event/producer"
	"github.com/chihaya/chihaya/tracker/models"
)

type driver struct{}

// Nop is a Producer for a BitTorrent tracker that does nothing.
type Nop struct{}

func (d *driver) New(cfg *config.DriverConfig) (producer.Producer, error) {
	return &Nop{}, nil
}

// Close returns nil.
func (n *Nop) Close() error {
	return nil
}

// LoadTorrents returns (nil, nil).
func (n *Nop) LoadTorrents(ids []uint64) ([]*models.Torrent, error) {
	return nil, nil
}

// LoadAllTorrents returns (nil, nil).
func (n *Nop) LoadAllTorrents() ([]*models.Torrent, error) {
	return nil, nil
}

// LoadUsers returns (nil, nil).
func (n *Nop) LoadUsers(ids []uint64) ([]*models.User, error) {
	return nil, nil
}

// LoadAllUsers returns (nil, nil).
func (n *Nop) LoadAllUsers(ids []uint64) ([]*models.User, error) {
	return nil, nil
}

// Init registers the nop driver.
func init() {
	producer.Register("nop", &driver{})
}
