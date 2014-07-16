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

	_ "github.com/chihaya/chihaya/drivers/backend/noop"
	_ "github.com/chihaya/chihaya/drivers/tracker/memory"
)

type params map[string]string

var infoHash = string([]byte{0x89, 0xd4, 0xbc, 0x52, 0x11, 0x16, 0xca, 0x1d, 0x42, 0xa2, 0xf3, 0x0d, 0x1f, 0x27, 0x4d, 0x94, 0xe4, 0x68, 0x1d, 0xaf})

func setupTracker(cfg *config.Config) (*httptest.Server, error) {
	tkr, err := NewTracker(cfg)
	if err != nil {
		return nil, err
	}
	return createServer(tkr, cfg)
}

func createServer(tkr *Tracker, cfg *config.Config) (*httptest.Server, error) {
	return httptest.NewServer(NewRouter(tkr, cfg)), nil
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
