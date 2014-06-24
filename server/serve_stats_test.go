// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chihaya/chihaya/config"
	_ "github.com/chihaya/chihaya/drivers/backend/mock"
	_ "github.com/chihaya/chihaya/drivers/tracker/mock"
)

func newTestServer() (*Server, error) {
	return New(&config.MockConfig)
}

func TestStats(t *testing.T) {
	s, err := newTestServer()
	if err != nil {
		t.Error(err)
	}

	r, err := http.NewRequest("GET", "127.0.0.1:80/stats", nil)
	if err != nil {
		t.Error(err)
	}

	w := httptest.NewRecorder()
	s.serveStats(w, r)

	if w.Code != 200 {
		t.Error("/stats did not return 200 OK")
	}

	if w.Header()["Content-Type"][0] != "application/json" {
		t.Error("/stats did not return JSON")
	}
}
