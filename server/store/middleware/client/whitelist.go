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
	tracker.RegisterAnnounceMiddleware("client_whitelist", whitelistAnnounceClient)
}

// PrefixClient is the prefix to be used for client peer IDs.
const PrefixClient = "c-"

// ErrNotWhitelistedClient is returned by an announce middleware if the
// announcing Client is not whitelisted.
var ErrNotWhitelistedClient = tracker.ClientError("client not whitelisted")

// whitelistAnnounceClient provides a middleware that only allows Clients to
// announce that are stored in the StringStore.
func whitelistAnnounceClient(next tracker.AnnounceHandler) tracker.AnnounceHandler {
	return func(cfg *chihaya.TrackerConfig, req *chihaya.AnnounceRequest, resp *chihaya.AnnounceResponse) error {
		whitelisted, err := store.MustGetStore().HasString(PrefixClient + clientid.New(string(req.PeerID[:])))
		if err != nil {
			return err
		} else if !whitelisted {
			return ErrNotWhitelistedClient
		}
		return next(cfg, req, resp)
	}
}
