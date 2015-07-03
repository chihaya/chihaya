// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package nop implements a Chihaya deltastore as a no-op. This is useful for
// running Chihaya when you do not want to use a delta store.
package nop

import (
	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/deltastore"
	"github.com/chihaya/chihaya/tracker/models"
)

type driver struct{}

// Nop is a delta store for Chihaya that does nothing.
type Nop struct{}

// New returns a new Chihaya backend driver that does nothing.
func (d *driver) New(cfg *config.DriverConfig) (deltastore.Conn, error) {
	return &Nop{}, nil
}

// Close returns nil.
func (n *Nop) Close() error {
	return nil
}

// Ping returns nil.
func (n *Nop) Ping() error {
	return nil
}

// RecordAnnounce returns nil.
func (n *Nop) RecordAnnounce(delta *models.AnnounceDelta) error {
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

// Init registers the nop driver as a backend for Chihaya.
func init() {
	deltastore.Register("nop", &driver{})
}
