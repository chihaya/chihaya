// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package client

import (
	"github.com/chihaya/chihaya"
	"github.com/chihaya/chihaya/server/store"
	"github.com/chihaya/chihaya/tracker"
)

func init() {
	tracker.RegisterAnnounceMiddleware("client_whitelist", whitelistAnnounceClient)
}

// whitelistAnnounceClient provides a middleware that only allows Clients to
// announce that are stored in a ClientStore.
func whitelistAnnounceClient(next tracker.AnnounceHandler) tracker.AnnounceHandler {
	return func(cfg *chihaya.TrackerConfig, req *chihaya.AnnounceRequest, resp *chihaya.AnnounceResponse) error {
		whitelisted, err := store.MustGetStore().FindClient(req.PeerID)

		if err != nil {
			return err
		} else if !whitelisted {
			return ErrBlockedClient
		}
		return next(cfg, req, resp)
	}
}
