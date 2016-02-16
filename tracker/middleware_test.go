// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.package middleware

package tracker

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/chihaya/chihaya"
	"github.com/chihaya/chihaya/config"
)

func testAnnounceMW1(next AnnounceHandler) AnnounceHandler {
	return func(cfg *config.TrackerConfig, req *chihaya.AnnounceRequest, resp *chihaya.AnnounceResponse) error {
		resp.IPv4Peers = append(resp.IPv4Peers, chihaya.Peer{
			Port: 1,
		})
		return next(cfg, req, resp)
	}
}

func testAnnounceMW2(next AnnounceHandler) AnnounceHandler {
	return func(cfg *config.TrackerConfig, req *chihaya.AnnounceRequest, resp *chihaya.AnnounceResponse) error {
		resp.IPv4Peers = append(resp.IPv4Peers, chihaya.Peer{
			Port: 2,
		})
		return next(cfg, req, resp)
	}
}

func testAnnounceMW3(next AnnounceHandler) AnnounceHandler {
	return func(cfg *config.TrackerConfig, req *chihaya.AnnounceRequest, resp *chihaya.AnnounceResponse) error {
		resp.IPv4Peers = append(resp.IPv4Peers, chihaya.Peer{
			Port: 3,
		})
		return next(cfg, req, resp)
	}
}

func TestAnnounceChain(t *testing.T) {
	var achain announceChain
	achain.Append(testAnnounceMW1)
	achain.Append(testAnnounceMW2)
	achain.Append(testAnnounceMW3)
	handler := achain.Handler()
	resp := &chihaya.AnnounceResponse{}
	err := handler(nil, &chihaya.AnnounceRequest{}, resp)
	assert.Nil(t, err, "the handler should not return an error")
	assert.Equal(t, resp.IPv4Peers, []chihaya.Peer{chihaya.Peer{Port: 1}, chihaya.Peer{Port: 2}, chihaya.Peer{Port: 3}}, "the list of peers added from the middleware should be in the same order.")
}
