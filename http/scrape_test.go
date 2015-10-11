// Copyright 2015 The Chihaya Authors. All rights reserved.
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
)

func TestPublicScrape(t *testing.T) {
	srv, err := setupTracker(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	scrapeParams := params{"info_hash": infoHash}

	// Add one seeder.
	peer := makePeerParams("peer1", true)
	announce(peer, srv)

	checkScrape(scrapeParams, makeScrapeResponse(1, 0, 0), srv, t)

	// Add another seeder.
	peer = makePeerParams("peer2", true)
	announce(peer, srv)

	checkScrape(scrapeParams, makeScrapeResponse(2, 0, 0), srv, t)

	// Add a leecher.
	peer = makePeerParams("peer3", false)
	announce(peer, srv)

	checkScrape(scrapeParams, makeScrapeResponse(2, 1, 0), srv, t)

	// Remove seeder.
	peer = makePeerParams("peer1", true)
	peer["event"] = "stopped"
	announce(peer, srv)

	checkScrape(scrapeParams, makeScrapeResponse(1, 1, 0), srv, t)

	// Complete torrent.
	peer = makePeerParams("peer3", true)
	peer["event"] = "complete"
	announce(peer, srv)

	checkScrape(scrapeParams, makeScrapeResponse(2, 0, 0), srv, t)
}

func makeScrapeResponse(seeders, leechers, downloaded int64) bencode.Dict {
	return bencode.Dict{
		"files": bencode.Dict{
			infoHash: bencode.Dict{
				"complete":   seeders,
				"incomplete": leechers,
				"downloaded": downloaded,
			},
		},
	}
}

func checkScrape(p params, expected interface{}, srv *httptest.Server, t *testing.T) bool {
	values := &url.Values{}
	for k, v := range p {
		values.Add(k, v)
	}

	response, err := http.Get(srv.URL + "/scrape?" + values.Encode())
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
