// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package server

import (
  "encoding/json"
  "net/http"
  "sync/atomic"
  "time"

  "github.com/pushrax/chihaya/config"
)

type stats struct {
  Uptime config.Duration `json:"uptime"`
  RPM    int64           `json:"req_per_min"`
}

func (s *Server) serveStats(w http.ResponseWriter, r *http.Request) {
  w.Header().Set("Content-Type", "application/json")

  stats, _ := json.Marshal(&stats{
    config.Duration{time.Now().Sub(s.startTime)},
    s.rpm,
  })

  length, _ := w.Write(stats)
  w.Header().Set("Content-Length", string(length))
  w.(http.Flusher).Flush()
}

func (s *Server) updateStats() {
  for _ = range time.NewTicker(time.Minute).C {
    s.rpm = s.deltaRequests
    atomic.StoreInt64(&s.deltaRequests, 0)
  }
}
