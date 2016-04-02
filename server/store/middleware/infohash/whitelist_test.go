// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package infohash

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/chihaya/chihaya"
	"github.com/chihaya/chihaya/tracker"
)

func TestWhitelistAnnounceMiddleware(t *testing.T) {
	var (
		achain tracker.AnnounceChain
		req    chihaya.AnnounceRequest
		resp   chihaya.AnnounceResponse
	)

	achain.Append(whitelistAnnounceInfohash)
	handler := achain.Handler()

	err := handler(nil, &req, &resp)
	assert.Equal(t, ErrBlockedInfohash, err)

	req.InfoHash = chihaya.InfoHash("def")
	err = handler(nil, &req, &resp)
	assert.Equal(t, ErrBlockedInfohash, err)

	req.InfoHash = chihaya.InfoHash("abc")
	err = handler(nil, &req, &resp)
	assert.Nil(t, err)
}

func TestWhitelistScrapeMiddlewareBlock(t *testing.T) {
	var (
		schain tracker.ScrapeChain
		req    chihaya.ScrapeRequest
		resp   chihaya.ScrapeResponse
	)

	mw, err := whitelistScrapeInfohash(chihaya.MiddlewareConfig{
		Name: "whitelist_infohash",
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

	req.InfoHashes = []chihaya.InfoHash{chihaya.InfoHash("abc")}
	err = handler(nil, &req, &resp)
	assert.Nil(t, err)
}

func TestWhitelistScrapeMiddlewareFilter(t *testing.T) {
	var (
		schain tracker.ScrapeChain
		req    chihaya.ScrapeRequest
		resp   chihaya.ScrapeResponse
	)

	mw, err := whitelistScrapeInfohash(chihaya.MiddlewareConfig{
		Name: "whitelist_infohash",
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
	assert.Equal(t, []chihaya.InfoHash{chihaya.InfoHash("abc")}, req.InfoHashes)

	req.InfoHashes = []chihaya.InfoHash{chihaya.InfoHash("abc")}
	err = handler(nil, &req, &resp)
	assert.Nil(t, err)
	assert.Equal(t, []chihaya.InfoHash{chihaya.InfoHash("abc")}, req.InfoHashes)
}

func TestZTearDown(t *testing.T) {
	srv.Stop()
}
