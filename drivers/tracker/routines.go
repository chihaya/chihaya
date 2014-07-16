// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package tracker

import (
	"time"

	"github.com/golang/glog"

	"github.com/chihaya/chihaya/config"
)

func purgeTorrents(p Pool, threshold time.Duration) {
	for _ = range time.NewTicker(time.Minute).C {
		before := time.Now().Add(-threshold)
		glog.V(0).Infof("Purging torrents before %s", before)

		conn, err := p.Get()

		if err != nil {
			glog.Error("Unable to get connection for a routine")
			continue
		}

		err = conn.PurgeInactiveTorrents(before)
		if err != nil {
			glog.Errorf("Error purging torrents: ", err)
		}
	}
}

func StartPurgingRoutines(p Pool, cfg *config.DriverConfig) error {
	if interval := cfg.Params["purge_after"]; interval != "" {
		threshold, err := time.ParseDuration(interval)
		if err != nil {
			return err
		}

		go purgeTorrents(p, threshold)
	}
	return nil
}
