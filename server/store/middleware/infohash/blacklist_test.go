// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package infohash

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/chihaya/chihaya"
	"github.com/chihaya/chihaya/server/store"
	"github.com/chihaya/chihaya/tracker"
)

type storeMock struct {
	strings map[string]struct{}
}

func (ss *storeMock) PutString(s string) error {
	ss.strings[s] = struct{}{}

	return nil
}

func (ss *storeMock) HasString(s string) (bool, error) {
	_, ok := ss.strings[s]

	return ok, nil
}

func (ss *storeMock) RemoveString(s string) error {
	delete(ss.strings, s)

	return nil
}

var mock store.StringStore = &storeMock{
	strings: make(map[string]struct{}),
}

var (
	ih1 = chihaya.InfoHash([20]byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	ih2 = chihaya.InfoHash([20]byte{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
)

func TestASetUp(t *testing.T) {
	mustGetStore = func() store.StringStore {
		return mock
	}

	mustGetStore().PutString(PrefixInfohash + string(ih1[:]))
}

func TestBlacklistAnnounceMiddleware(t *testing.T) {
	var (
		achain tracker.AnnounceChain
		req    chihaya.AnnounceRequest
		resp   chihaya.AnnounceResponse
	)

	achain.Append(blacklistAnnounceInfohash)
	handler := achain.Handler()

	err := handler(nil, &req, &resp)
	assert.Nil(t, err)

	req.InfoHash = chihaya.InfoHash(ih1)
	err = handler(nil, &req, &resp)
	assert.Equal(t, ErrBlockedInfohash, err)

	req.InfoHash = chihaya.InfoHash(ih2)
	err = handler(nil, &req, &resp)
	assert.Nil(t, err)
}

func TestBlacklistScrapeMiddlewareBlock(t *testing.T) {
	var (
		schain tracker.ScrapeChain
		req    chihaya.ScrapeRequest
		resp   chihaya.ScrapeResponse
	)

	mw, err := blacklistScrapeInfohash(chihaya.MiddlewareConfig{
		Name: "blacklist_infohash",
		Config: Config{
			Mode: ModeBlock,
		},
	})
	assert.Nil(t, err)
	schain.Append(mw)
	handler := schain.Handler()

	err = handler(nil, &req, &resp)
	assert.Nil(t, err)

	req.InfoHashes = []chihaya.InfoHash{chihaya.InfoHash(ih1), chihaya.InfoHash(ih2)}
	err = handler(nil, &req, &resp)
	assert.Equal(t, ErrBlockedInfohash, err)

	req.InfoHashes = []chihaya.InfoHash{chihaya.InfoHash(ih2)}
	err = handler(nil, &req, &resp)
	assert.Nil(t, err)
}

func TestBlacklistScrapeMiddlewareFilter(t *testing.T) {
	var (
		schain tracker.ScrapeChain
		req    chihaya.ScrapeRequest
		resp   chihaya.ScrapeResponse
	)

	mw, err := blacklistScrapeInfohash(chihaya.MiddlewareConfig{
		Name: "blacklist_infohash",
		Config: Config{
			Mode: ModeFilter,
		},
	})
	assert.Nil(t, err)
	schain.Append(mw)
	handler := schain.Handler()

	err = handler(nil, &req, &resp)
	assert.Nil(t, err)

	req.InfoHashes = []chihaya.InfoHash{chihaya.InfoHash(ih1), chihaya.InfoHash(ih2)}
	err = handler(nil, &req, &resp)
	assert.Nil(t, err)
	assert.Equal(t, []chihaya.InfoHash{chihaya.InfoHash(ih2)}, req.InfoHashes)

	req.InfoHashes = []chihaya.InfoHash{chihaya.InfoHash(ih2)}
	err = handler(nil, &req, &resp)
	assert.Nil(t, err)
}
