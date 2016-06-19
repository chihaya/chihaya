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
		closed:  make(chan struct{}),
	}, nil
}

type stringStore struct {
	strings map[string]struct{}
	closed  chan struct{}
	sync.RWMutex
}

var _ store.StringStore = &stringStore{}

func (ss *stringStore) PutString(s string) error {
	ss.Lock()
	defer ss.Unlock()

	select {
	case <-ss.closed:
		panic("attempted to interact with stopped store")
	default:
	}

	ss.strings[s] = struct{}{}

	return nil
}

func (ss *stringStore) HasString(s string) (bool, error) {
	ss.RLock()
	defer ss.RUnlock()

	select {
	case <-ss.closed:
		panic("attempted to interact with stopped store")
	default:
	}

	_, ok := ss.strings[s]

	return ok, nil
}

func (ss *stringStore) RemoveString(s string) error {
	ss.Lock()
	defer ss.Unlock()

	select {
	case <-ss.closed:
		panic("attempted to interact with stopped store")
	default:
	}

	if _, ok := ss.strings[s]; !ok {
		return store.ErrResourceDoesNotExist
	}

	delete(ss.strings, s)

	return nil
}

func (ss *stringStore) Stop() <-chan error {
	toReturn := make(chan error)
	go func() {
		ss.Lock()
		defer ss.Unlock()
		ss.strings = make(map[string]struct{})
		close(ss.closed)
		close(toReturn)
	}()
	return toReturn
}
