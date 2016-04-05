// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package client

import (
	"github.com/chihaya/chihaya"
	"github.com/chihaya/chihaya/pkg/clientid"
	"github.com/chihaya/chihaya/server/store"
	"github.com/chihaya/chihaya/tracker"
)

func init() {
	tracker.RegisterAnnounceMiddleware("client_blacklist", blacklistAnnounceClient)
}

// ErrBlacklistedClient is returned by an announce middleware if the announcing
// Client is blacklisted.
var ErrBlacklistedClient = tracker.ClientError("client blacklisted")

// blacklistAnnounceClient provides a middleware that only allows Clients to
// announce that are not stored in the StringStore.
func blacklistAnnounceClient(next tracker.AnnounceHandler) tracker.AnnounceHandler {
	return func(cfg *chihaya.TrackerConfig, req *chihaya.AnnounceRequest, resp *chihaya.AnnounceResponse) error {
		blacklisted, err := store.MustGetStore().HasString(PrefixClient + clientid.New(string(req.PeerID)))
		if err != nil {
			return err
		} else if blacklisted {
			return ErrBlacklistedClient
		}
		return next(cfg, req, resp)
	}
}
