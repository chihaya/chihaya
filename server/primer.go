// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package server

import (
	"github.com/chihaya/chihaya/storage/backend"
	"github.com/chihaya/chihaya/storage/tracker"
)

// Primer represents a function that can prime storage with data.
type Primer func(tracker.Pool, backend.Conn) error

// Prime executes a priming function on the server.
func (s *Server) Prime(p Primer) error {
	return p(s.trackerPool, s.backendConn)
}
