// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package memory

import (
	"sync"

	"github.com/chihaya/chihaya/pkg/clientid"
	"github.com/chihaya/chihaya/server/store"
)

func init() {
	store.RegisterClientStoreDriver("memory", &clientStoreDriver{})
}

type clientStoreDriver struct{}

func (d *clientStoreDriver) New(cfg *store.Config) (store.ClientStore, error) {
	return &clientStore{
		clientIDs: make(map[string]struct{}),
	}, nil
}

type clientStore struct {
	clientIDs map[string]struct{}
	sync.RWMutex
}

var _ store.ClientStore = &clientStore{}

func (s *clientStore) CreateClient(clientID string) error {
	s.Lock()
	defer s.Unlock()

	s.clientIDs[clientID] = struct{}{}

	return nil
}

func (s *clientStore) FindClient(peerID string) (bool, error) {
	clientID := clientid.New(peerID)
	s.RLock()
	defer s.RUnlock()

	_, ok := s.clientIDs[clientID]

	return ok, nil
}

func (s *clientStore) DeleteClient(clientID string) error {
	s.Lock()
	defer s.Unlock()

	delete(s.clientIDs, clientID)

	return nil
}
