// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package memory

import (
	"sync"

	"github.com/chihaya/chihaya/server/store"
)

func init() {
	store.RegisterStringStoreDriver("memory", &stringStoreDriver{})
}

type stringStoreDriver struct{}

func (d *stringStoreDriver) New(_ *store.DriverConfig) (store.StringStore, error) {
	return &stringStore{
		strings: make(map[string]struct{}),
	}, nil
}

type stringStore struct {
	strings map[string]struct{}
	sync.RWMutex
}

var _ store.StringStore = &stringStore{}

func (ss *stringStore) PutString(s string) error {
	ss.Lock()
	defer ss.Unlock()

	ss.strings[s] = struct{}{}

	return nil
}

func (ss *stringStore) HasString(s string) (bool, error) {
	ss.RLock()
	defer ss.RUnlock()

	_, ok := ss.strings[s]

	return ok, nil
}

func (ss *stringStore) RemoveString(s string) error {
	ss.Lock()
	defer ss.Unlock()

	if _, ok := ss.strings[s]; !ok {
		return store.ErrResourceDoesNotExist
	}

	delete(ss.strings, s)

	return nil
}
