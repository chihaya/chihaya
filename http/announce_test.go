// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package http

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"github.com/chihaya/bencode"
	"github.com/chihaya/chihaya/config"

	_ "github.com/chihaya/chihaya/drivers/backend/noop"
	_ "github.com/chihaya/chihaya/drivers/tracker/memory"
)

type params map[string]string

const infoHash = "%89%d4%bcR%11%16%ca%1dB%a2%f3%0d%1f%27M%94%e4h%1d%af"

func TestPublicAnnounce(t *testing.T) {
	srv, _ := setupTracker(&config.DefaultConfig)
	defer srv.Close()

	// Add one seeder.
	peer := makePeerParams("peer1", true)
	expected := makeResponse(1, 0, bencode.List{})
	checkResponse(peer, expected, srv, t)

	// Add another seeder.
	peer = makePeerParams("peer2", true)
	expected = makeResponse(2, 0, bencode.List{})
	checkResponse(peer, expected, srv, t)

	// Add a leecher.
	peer = makePeerParams("peer3", false)
	expected = makeResponse(2, 1, bencode.List{
		makePeerResponse("peer1"),
		makePeerResponse("peer2"),
	})
	checkResponse(peer, expected, srv, t)

	// Remove seeder.
	peer = makePeerParams("peer1", true)
	peer["event"] = "stopped"
	expected = makeResponse(1, 1, nil)
	checkResponse(peer, expected, srv, t)

	// Check seeders.
	peer = makePeerParams("peer3", false)
	expected = makeResponse(1, 1, bencode.List{
		makePeerResponse("peer2"),
	})
	checkResponse(peer, expected, srv, t)
}

func makePeerParams(id string, seed bool) params {
	left := "1"
	if seed {
		left = "0"
	}

	return params{
		"info_hash":  infoHash,
		"peer_id":    id,
		"port":       "1234",
		"uploaded":   "0",
		"downloaded": "0",
		"left":       left,
		"compact":    "0",
		"numwant":    "50",
	}
}

func makePeerResponse(id string) bencode.Dict {
	return bencode.Dict{
		"ip":      "127.0.0.1",
		"peer id": id,
		"port":    int64(1234),
	}
}

func makeResponse(seeders, leechers int64, peers bencode.List) bencode.Dict {
	dict := bencode.Dict{
		"complete":     seeders,
		"incomplete":   leechers,
		"interval":     int64(1800),
		"min interval": int64(900),
	}

	if peers != nil {
		dict["peers"] = peers
	}
	return dict
}

func checkResponse(p params, expected interface{}, srv *httptest.Server, t *testing.T) bool {
	values := &url.Values{}
	for k, v := range p {
		values.Add(k, v)
	}

	response, err := http.Get(srv.URL + "/announce?" + values.Encode())
	if err != nil {
		t.Error(err)
		return false
	}

	body, err := ioutil.ReadAll(response.Body)
	response.Body.Close()

	if err != nil {
		t.Error(err)
		return false
	}

	got, err := bencode.Unmarshal(body)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("\ngot:    %#v\nwanted: %#v", got, expected)
		return false
	}
	return true
}

func setupTracker(cfg *config.Config) (*httptest.Server, error) {
	tkr, err := NewTracker(cfg)
	if err != nil {
		return nil, err
	}

	return httptest.NewServer(setupRoutes(tkr, cfg)), nil
}
