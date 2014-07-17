// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package http

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"

	"github.com/chihaya/bencode"
	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/tracker"

	_ "github.com/chihaya/chihaya/drivers/backend/noop"
	_ "github.com/chihaya/chihaya/tracker/memory"
)

type params map[string]string

var infoHash = string([]byte{0x89, 0xd4, 0xbc, 0x52, 0x11, 0x16, 0xca, 0x1d, 0x42, 0xa2, 0xf3, 0x0d, 0x1f, 0x27, 0x4d, 0x94, 0xe4, 0x68, 0x1d, 0xaf})

func setupTracker(cfg *config.Config) (*httptest.Server, error) {
	tkr, err := tracker.New(cfg)
	if err != nil {
		return nil, err
	}
	return createServer(tkr, cfg)
}

func createServer(tkr *tracker.Tracker, cfg *config.Config) (*httptest.Server, error) {
	srv := &Server{
		config:  cfg,
		tracker: tkr,
	}
	return httptest.NewServer(NewRouter(srv)), nil
}

func announce(p params, srv *httptest.Server) ([]byte, error) {
	values := &url.Values{}
	for k, v := range p {
		values.Add(k, v)
	}

	body, _, err := fetchPath(srv.URL + "/announce?" + values.Encode())
	return body, err
}

func fetchPath(path string) ([]byte, int, error) {
	response, err := http.Get(path)
	if err != nil {
		return nil, 0, err
	}

	body, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	return body, response.StatusCode, err
}

type peerList bencode.List

func (p peerList) Len() int {
	return len(p)
}

func (p peerList) Less(i, j int) bool {
	return p[i].(bencode.Dict)["peer id"].(string) < p[j].(bencode.Dict)["peer id"].(string)
}

func (p peerList) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func sortPeersInResponse(dict bencode.Dict) {
	if peers, ok := dict["peers"].(bencode.List); ok {
		sort.Stable(peerList(peers))
	}
}
