// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package ip

import (
	"github.com/chihaya/chihaya"
	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/server/store"
	"github.com/chihaya/chihaya/tracker"
)

func init() {
	tracker.RegisterAnnounceMiddleware("client_blacklist", blacklistAnnounceClient)
}

// ErrBlockedClient is returned by an announce middleware if the announcing
// Client is disallowed.
var ErrBlockedClient = tracker.ClientError("disallowed client")

// blacklistAnnounceClient provides a middleware that only allows Clients to
// announce that are not stored in a ClientStore.
func blacklistAnnounceClient(next tracker.AnnounceHandler) tracker.AnnounceHandler {
	return func(cfg *config.TrackerConfig, req *chihaya.AnnounceRequest, resp *chihaya.AnnounceResponse) error {
		blacklisted, err := store.MustGetStore().FindClient(req.PeerID)

		if err != nil {
			return err
		} else if blacklisted {
			return ErrBlockedClient
		}
		return next(cfg, req, resp)
	}
}
