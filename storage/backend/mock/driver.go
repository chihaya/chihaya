// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package mock implements the storage interface for a BitTorrent tracker's
// backend storage. It can be used in production, but isn't recommended.
// Stored values will not persist if the tracker is restarted.
package mock

import (
	"sync"

	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/storage"
	"github.com/chihaya/chihaya/storage/backend"
)

type driver struct{}

// Mock is a concrete implementation of the backend.Conn interface (plus some
// debugging methods) that stores deltas in memory.
type Mock struct {
	deltaHistory  []*backend.AnnounceDelta
	deltaHistoryM sync.RWMutex
}

func (d *driver) New(conf *config.DataStore) backend.Conn {
	return &Mock{}
}

// Start returns nil.
func (m *Mock) Start() error {
	return nil
}

// Close returns nil.
func (m *Mock) Close() error {
	return nil
}

// RecordAnnounce adds a delta to the history.
func (m *Mock) RecordAnnounce(delta *backend.AnnounceDelta) error {
	m.deltaHistoryM.Lock()
	defer m.deltaHistoryM.Unlock()

	m.deltaHistory = append(m.deltaHistory, delta)

	return nil
}

// DeltaHistory safely copies and returns the history of recorded deltas.
func (m *Mock) DeltaHistory() []backend.AnnounceDelta {
	m.deltaHistoryM.Lock()
	defer m.deltaHistoryM.Unlock()

	cp := make([]backend.AnnounceDelta, len(m.deltaHistory))
	for index, delta := range m.deltaHistory {
		cp[index] = *delta
	}

	return cp
}

// LoadTorrents returns (nil, nil).
func (m *Mock) LoadTorrents(ids []uint64) ([]*storage.Torrent, error) {
	return nil, nil
}

// LoadAllTorrents returns (nil, nil).
func (m *Mock) LoadAllTorrents() ([]*storage.Torrent, error) {
	return nil, nil
}

// LoadUsers returns (nil, nil).
func (m *Mock) LoadUsers(ids []uint64) ([]*storage.User, error) {
	return nil, nil
}

// LoadAllUsers returns (nil, nil).
func (m *Mock) LoadAllUsers(ids []uint64) ([]*storage.User, error) {
	return nil, nil
}

func init() {
	backend.Register("mock", &driver{})
}
