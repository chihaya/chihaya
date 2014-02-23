// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package mock implements the storage interface for a BitTorrent tracker's
// backend storage. It can be used in production, but isn't recommended.
// Stored values will not persist if the tracker is restarted.
package mock

import (
	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/storage"
	"github.com/chihaya/chihaya/storage/backend"
)

type driver struct{}

type mock struct{}

func (d *driver) New(conf *config.DataStore) backend.Conn {
	return &mock{}
}

func (m *mock) Start() error {
	return nil
}

func (m *mock) Close() error {
	return nil
}

func (m *mock) RecordAnnounce(delta *backend.AnnounceDelta) error {
	return nil
}

func (m *mock) LoadTorrents(ids []uint64) ([]*storage.Torrent, error) {
	return nil, nil
}

func (m *mock) LoadAllTorrents() ([]*storage.Torrent, error) {
	return nil, nil
}

func (m *mock) LoadUsers(ids []uint64) ([]*storage.User, error) {
	return nil, nil
}

func (m *mock) LoadAllUsers(ids []uint64) ([]*storage.User, error) {
	return nil, nil
}

func init() {
	backend.Register("mock", &driver{})
}
