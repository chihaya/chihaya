// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package noop implements a Chihaya backend storage driver as a no-op. This is
// useful for running Chihaya as a public tracker.
package noop

import (
	"github.com/chihaya/chihaya/backend"
	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/tracker/models"
)

type driver struct{}

type NoOp struct{}

// New returns a new Chihaya backend driver that does nothing.
func (d *driver) New(cfg *config.DriverConfig) (backend.Conn, error) {
	return &NoOp{}, nil
}

// Close returns nil.
func (n *NoOp) Close() error {
	return nil
}

// RecordAnnounce returns nil.
func (n *NoOp) RecordAnnounce(delta *models.AnnounceDelta) error {
	return nil
}

// LoadTorrents returns (nil, nil).
func (n *NoOp) LoadTorrents(ids []uint64) ([]*models.Torrent, error) {
	return nil, nil
}

// LoadAllTorrents returns (nil, nil).
func (n *NoOp) LoadAllTorrents() ([]*models.Torrent, error) {
	return nil, nil
}

// LoadUsers returns (nil, nil).
func (n *NoOp) LoadUsers(ids []uint64) ([]*models.User, error) {
	return nil, nil
}

// LoadAllUsers returns (nil, nil).
func (n *NoOp) LoadAllUsers(ids []uint64) ([]*models.User, error) {
	return nil, nil
}

func init() {
	backend.Register("noop", &driver{})
}
