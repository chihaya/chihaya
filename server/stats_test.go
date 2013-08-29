// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package server

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/pushrax/chihaya/config"

	_ "github.com/pushrax/chihaya/cache/redis"
	_ "github.com/pushrax/chihaya/storage/batter"
)

func TestStats(t *testing.T) {
	testConfig, err := config.Open(os.ExpandEnv("$GOPATH/src/github.com/pushrax/chihaya/config/example.json"))
	if err != nil {
		t.Error(err)
	}
	fmt.Println(testConfig)
	s, err := New(testConfig)
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
