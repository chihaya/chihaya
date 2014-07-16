// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package http

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/chihaya/chihaya/config"

	_ "github.com/chihaya/chihaya/drivers/backend/noop"
	_ "github.com/chihaya/chihaya/drivers/tracker/memory"
)

type params map[string]string

const infoHash = "%89%d4%bcR%11%16%ca%1dB%a2%f3%0d%1f%27M%94%e4h%1d%af"

func setupTracker(cfg *config.Config) (*httptest.Server, error) {
	tkr, err := NewTracker(cfg)
	if err != nil {
		return nil, err
	}
	return httptest.NewServer(setupRoutes(tkr, cfg)), nil
}

func announce(p params, srv *httptest.Server) ([]byte, error) {
	values := &url.Values{}
	for k, v := range p {
		values.Add(k, v)
	}

	response, err := http.Get(srv.URL + "/announce?" + values.Encode())
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	return body, err
}
