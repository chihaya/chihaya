// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/pushrax/chihaya/config"

	_ "github.com/pushrax/chihaya/storage/tracker/redis"
	_ "github.com/pushrax/chihaya/storage/web/batter"
)

func newTestServer() (*Server, error) {
	testConfig, err := config.Open(os.Getenv("TESTCONFIGPATH"))
	if err != nil {
		return nil, err
	}

	s, err := New(testConfig)
	if err != nil {
		return nil, err
	}

	return s, nil
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
		t.Error(errors.New("/stats did not return HTTP 200"))
	}
}
