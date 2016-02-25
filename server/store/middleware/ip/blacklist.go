// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package ip

import (
	"net"

	"github.com/chihaya/chihaya"
	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/errors"
	"github.com/chihaya/chihaya/server/store"
	"github.com/chihaya/chihaya/tracker"
)

func init() {
	tracker.RegisterAnnounceMiddleware("IPBlacklist", blacklistAnnounceIP)
}

// ErrBlockedIP is returned by an announce middleware if any of the announcing
// IPs is disallowed.
var ErrBlockedIP = errors.NewMessage("disallowed IP address")

// blacklistAnnounceIP provides a middleware that only allows IPs to announce
// that are not stored in an IPStore.
func blacklistAnnounceIP(next tracker.AnnounceHandler) tracker.AnnounceHandler {
	return func(cfg *config.TrackerConfig, req *chihaya.AnnounceRequest, resp *chihaya.AnnounceResponse) (err error) {
		blacklisted := false
		storage := store.MustGetStore()

		// We have to check explicitly if they are present, because someone
		// could have added a <nil> net.IP to the store.
		if req.IPv6 != nil && req.IPv4 != nil {
			blacklisted, err = storage.HasAnyIP([]net.IP{req.IPv4, req.IPv6})
		} else if req.IPv4 != nil {
			blacklisted, err = storage.HasIP(req.IPv4)
		} else {
			blacklisted, err = storage.HasIP(req.IPv6)
		}

		if err != nil {
			return err
		} else if blacklisted {
			return ErrBlockedIP
		}
		return next(cfg, req, resp)
	}
}
