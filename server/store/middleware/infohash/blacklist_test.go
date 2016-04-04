// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package infohash

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/chihaya/chihaya"
	"github.com/chihaya/chihaya/server"
	"github.com/chihaya/chihaya/server/store"
	"github.com/chihaya/chihaya/tracker"

	_ "github.com/chihaya/chihaya/server/store/memory"
)

var srv server.Server

func TestASetUp(t *testing.T) {
	serverConfig := chihaya.ServerConfig{
		Name: "store",
		Config: store.Config{
			Addr: "localhost:6880",
			StringStore: store.DriverConfig{
				Name: "memory",
			},
			IPStore: store.DriverConfig{
				Name: "memory",
			},
			PeerStore: store.DriverConfig{
				Name: "memory",
			},
		},
	}

	var err error
	srv, err = server.New(&serverConfig, &tracker.Tracker{})
	assert.Nil(t, err)
	srv.Start()

	store.MustGetStore().PutString(store.PrefixInfohash + "abc")
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

	req.InfoHash = chihaya.InfoHash("abc")
	err = handler(nil, &req, &resp)
	assert.Equal(t, ErrBlockedInfohash, err)

	req.InfoHash = chihaya.InfoHash("def")
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

	req.InfoHashes = []chihaya.InfoHash{chihaya.InfoHash("abc"), chihaya.InfoHash("def")}
	err = handler(nil, &req, &resp)
	assert.Equal(t, ErrBlockedInfohash, err)

	req.InfoHashes = []chihaya.InfoHash{chihaya.InfoHash("def")}
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

	req.InfoHashes = []chihaya.InfoHash{chihaya.InfoHash("abc"), chihaya.InfoHash("def")}
	err = handler(nil, &req, &resp)
	assert.Nil(t, err)
	assert.Equal(t, []chihaya.InfoHash{chihaya.InfoHash("def")}, req.InfoHashes)

	req.InfoHashes = []chihaya.InfoHash{chihaya.InfoHash("def")}
	err = handler(nil, &req, &resp)
	assert.Nil(t, err)
}
