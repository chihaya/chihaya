// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package tracker

import (
	"time"

	"github.com/golang/glog"
)

func PurgeInactivePeers(p Pool, purgeEmptyTorrents bool, threshold, interval time.Duration) {
	for _ = range time.NewTicker(interval).C {
		before := time.Now().Add(-threshold)
		glog.V(0).Infof("Purging peers with no announces since %s", before)

		conn, err := p.Get()

		if err != nil {
			glog.Error("Unable to get connection for a routine")
			continue
		}

		err = conn.PurgeInactivePeers(purgeEmptyTorrents, before)
		if err != nil {
			glog.Errorf("Error purging torrents: %s", err)
		}
	}
}
