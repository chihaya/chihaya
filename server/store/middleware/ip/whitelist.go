// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package ip

import (
	"net"

	"github.com/chihaya/chihaya"
	"github.com/chihaya/chihaya/server/store"
	"github.com/chihaya/chihaya/tracker"
)

func init() {
	tracker.RegisterAnnounceMiddleware("ip_whitelist", whitelistAnnounceIP)
}

// whitelistAnnounceIP provides a middleware that only allows IPs to announce
// that are stored in an IPStore.
func whitelistAnnounceIP(next tracker.AnnounceHandler) tracker.AnnounceHandler {
	return func(cfg *chihaya.TrackerConfig, req *chihaya.AnnounceRequest, resp *chihaya.AnnounceResponse) (err error) {
		whitelisted := false
		storage := store.MustGetStore()

		// We have to check explicitly if they are present, because someone
		// could have added a <nil> net.IP to the store.
		if req.IPv4 != nil && req.IPv6 != nil {
			whitelisted, err = storage.HasAllIPs([]net.IP{req.IPv4, req.IPv6})
		} else if req.IPv4 != nil {
			whitelisted, err = storage.HasIP(req.IPv4)
		} else {
			whitelisted, err = storage.HasIP(req.IPv6)
		}

		if err != nil {
			return err
		} else if !whitelisted {
			return ErrBlockedIP
		}
		return next(cfg, req, resp)
	}
}
