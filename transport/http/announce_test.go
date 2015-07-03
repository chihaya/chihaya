// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package http

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/chihaya/bencode"
	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/tracker"
	"github.com/chihaya/chihaya/tracker/models"
)

func TestPublicAnnounce(t *testing.T) {
	srv, err := setupTracker(&config.DefaultConfig)
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	peer1 := makePeerParams("peer1", true)
	peer2 := makePeerParams("peer2", true)
	peer3 := makePeerParams("peer3", false)

	peer1["event"] = "started"
	expected := makeResponse(1, 0, peer1)
	checkAnnounce(peer1, expected, srv, t)

	expected = makeResponse(2, 0, peer2)
	checkAnnounce(peer2, expected, srv, t)

	expected = makeResponse(2, 1, peer1, peer2)
	checkAnnounce(peer3, expected, srv, t)

	peer1["event"] = "stopped"
	expected = makeResponse(1, 1, nil)
	checkAnnounce(peer1, expected, srv, t)

	expected = makeResponse(1, 1, peer2)
	checkAnnounce(peer3, expected, srv, t)
}

func TestTorrentPurging(t *testing.T) {
	cfg := config.DefaultConfig
	srv, err := setupTracker(&cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	torrentAPIPath := srv.URL + "/torrents/" + url.QueryEscape(infoHash)

	// Add one seeder.
	peer := makePeerParams("peer1", true)
	announce(peer, srv)

	_, status, err := fetchPath(torrentAPIPath)
	if err != nil {
		t.Fatal(err)
	} else if status != http.StatusOK {
		t.Fatalf("expected torrent to exist (got %s)", http.StatusText(status))
	}

	// Remove seeder.
	peer = makePeerParams("peer1", true)
	peer["event"] = "stopped"
	announce(peer, srv)

	_, status, err = fetchPath(torrentAPIPath)
	if err != nil {
		t.Fatal(err)
	} else if status != http.StatusNotFound {
		t.Fatalf("expected torrent to have been purged (got %s)", http.StatusText(status))
	}
}

func TestStalePeerPurging(t *testing.T) {
	cfg := config.DefaultConfig
	cfg.Announce = config.Duration{
		Duration: 10 * time.Millisecond,
	}

	srv, err := setupTracker(&cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	torrentAPIPath := srv.URL + "/torrents/" + url.QueryEscape(infoHash)

	// Add one seeder.
	peer1 := makePeerParams("peer1", true)
	announce(peer1, srv)

	_, status, err := fetchPath(torrentAPIPath)
	if err != nil {
		t.Fatal(err)
	} else if status != http.StatusOK {
		t.Fatalf("expected torrent to exist (got %s)", http.StatusText(status))
	}

	// Add a leecher.
	peer2 := makePeerParams("peer2", false)
	expected := makeResponse(1, 1, peer1)
	expected["interval"] = int64(0)
	checkAnnounce(peer2, expected, srv, t)

	// Let them both expire.
	time.Sleep(30 * time.Millisecond)

	_, status, err = fetchPath(torrentAPIPath)
	if err != nil {
		t.Fatal(err)
	} else if status != http.StatusNotFound {
		t.Fatalf("expected torrent to have been purged (got %s)", http.StatusText(status))
	}
}

func TestPrivateAnnounce(t *testing.T) {
	cfg := config.DefaultConfig
	cfg.PrivateEnabled = true

	tkr, err := tracker.New(&cfg)
	if err != nil {
		t.Fatal(err)
	}

	loadPrivateTestData(tkr)

	srv, err := createServer(tkr, &cfg)
	if err != nil {
		t.Fatal(err)
	}

	defer srv.Close()
	baseURL := srv.URL

	peer1 := makePeerParams("-TR2820-peer1", false)
	peer2 := makePeerParams("-TR2820-peer2", false)
	peer3 := makePeerParams("-TR2820-peer3", true)

	expected := makeResponse(0, 1, peer1)
	srv.URL = baseURL + "/users/vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv1"
	checkAnnounce(peer1, expected, srv, t)

	expected = makeResponse(0, 2, peer1)
	srv.URL = baseURL + "/users/vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv2"
	checkAnnounce(peer2, expected, srv, t)

	expected = makeResponse(1, 2, peer1, peer2)
	srv.URL = baseURL + "/users/vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv3"
	checkAnnounce(peer3, expected, srv, t)

	expected = makeResponse(1, 2, peer2, peer3)
	srv.URL = baseURL + "/users/vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv1"
	checkAnnounce(peer1, expected, srv, t)
}

func TestPreferredSubnet(t *testing.T) {
	cfg := config.DefaultConfig
	cfg.PreferredSubnet = true
	cfg.PreferredIPv4Subnet = 8
	cfg.PreferredIPv6Subnet = 16
	cfg.DualStackedPeers = false

	srv, err := setupTracker(&cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	peerA1 := makePeerParams("peerA1", false, "44.0.0.1")
	peerA2 := makePeerParams("peerA2", false, "44.0.0.2")
	peerA3 := makePeerParams("peerA3", false, "44.0.0.3")
	peerA4 := makePeerParams("peerA4", false, "44.0.0.4")
	peerB1 := makePeerParams("peerB1", false, "45.0.0.1")
	peerB2 := makePeerParams("peerB2", false, "45.0.0.2")
	peerC1 := makePeerParams("peerC1", false, "fc01::1")
	peerC2 := makePeerParams("peerC2", false, "fc01::2")
	peerC3 := makePeerParams("peerC3", false, "fc01::3")
	peerD1 := makePeerParams("peerD1", false, "fc02::1")
	peerD2 := makePeerParams("peerD2", false, "fc02::2")

	expected := makeResponse(0, 1, peerA1)
	checkAnnounce(peerA1, expected, srv, t)

	expected = makeResponse(0, 2, peerA1)
	checkAnnounce(peerA2, expected, srv, t)

	expected = makeResponse(0, 3, peerA1, peerA2)
	checkAnnounce(peerB1, expected, srv, t)

	peerB2["numwant"] = "1"
	expected = makeResponse(0, 4, peerB1)
	checkAnnounce(peerB2, expected, srv, t)
	checkAnnounce(peerB2, expected, srv, t)

	peerA3["numwant"] = "2"
	expected = makeResponse(0, 5, peerA1, peerA2)
	checkAnnounce(peerA3, expected, srv, t)
	checkAnnounce(peerA3, expected, srv, t)

	peerA4["numwant"] = "3"
	expected = makeResponse(0, 6, peerA1, peerA2, peerA3)
	checkAnnounce(peerA4, expected, srv, t)
	checkAnnounce(peerA4, expected, srv, t)

	expected = makeResponse(0, 7, peerA1, peerA2, peerA3, peerA4, peerB1, peerB2)
	checkAnnounce(peerC1, expected, srv, t)

	peerC2["numwant"] = "1"
	expected = makeResponse(0, 8, peerC1)
	checkAnnounce(peerC2, expected, srv, t)
	checkAnnounce(peerC2, expected, srv, t)

	peerC3["numwant"] = "2"
	expected = makeResponse(0, 9, peerC1, peerC2)
	checkAnnounce(peerC3, expected, srv, t)
	checkAnnounce(peerC3, expected, srv, t)

	expected = makeResponse(0, 10, peerA1, peerA2, peerA3, peerA4, peerB1, peerB2, peerC1, peerC2, peerC3)
	checkAnnounce(peerD1, expected, srv, t)

	peerD2["numwant"] = "1"
	expected = makeResponse(0, 11, peerD1)
	checkAnnounce(peerD2, expected, srv, t)
	checkAnnounce(peerD2, expected, srv, t)
}

func TestCompactAnnounce(t *testing.T) {
	srv, err := setupTracker(&config.DefaultConfig)
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	compact := "\xff\x09\x7f\x05\x04\xd2"
	ip := "255.9.127.5" // Use the same IP for all of them so we don't have to worry about order.

	peer1 := makePeerParams("peer1", false, ip)
	peer1["compact"] = "1"

	peer2 := makePeerParams("peer2", false, ip)
	peer2["compact"] = "1"

	peer3 := makePeerParams("peer3", false, ip)
	peer3["compact"] = "1"

	expected := makeResponse(0, 1)
	expected["peers"] = compact
	checkAnnounce(peer1, expected, srv, t)

	expected = makeResponse(0, 2)
	expected["peers"] = compact
	checkAnnounce(peer2, expected, srv, t)

	expected = makeResponse(0, 3)
	expected["peers"] = compact + compact
	checkAnnounce(peer3, expected, srv, t)
}

func makePeerParams(id string, seed bool, extra ...string) params {
	left := "1"
	if seed {
		left = "0"
	}

	ip := "10.0.0.1"
	if len(extra) >= 1 {
		ip = extra[0]
	}

	return params{
		"info_hash":  infoHash,
		"peer_id":    id,
		"ip":         ip,
		"port":       "1234",
		"uploaded":   "0",
		"downloaded": "0",
		"left":       left,
		"compact":    "0",
		"numwant":    "50",
	}
}

func peerFromParams(peer params) bencode.Dict {
	port, _ := strconv.ParseInt(peer["port"], 10, 64)

	return bencode.Dict{
		"peer id": peer["peer_id"],
		"ip":      peer["ip"],
		"port":    port,
	}
}

func makeResponse(seeders, leechers int64, peers ...params) bencode.Dict {
	dict := bencode.Dict{
		"complete":     seeders,
		"incomplete":   leechers,
		"interval":     int64(1800),
		"min interval": int64(900),
	}

	if !(len(peers) == 1 && peers[0] == nil) {
		peerList := bencode.List{}
		for _, peer := range peers {
			peerList = append(peerList, peerFromParams(peer))
		}
		dict["peers"] = peerList
	}
	return dict
}

func checkAnnounce(p params, expected interface{}, srv *httptest.Server, t *testing.T) bool {
	body, err := announce(p, srv)
	if err != nil {
		t.Error(err)
		return false
	}

	if e, ok := expected.(bencode.Dict); ok {
		sortPeersInResponse(e)
	}

	got, err := bencode.Unmarshal(body)
	if e, ok := got.(bencode.Dict); ok {
		sortPeersInResponse(e)
	}

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("\ngot:    %#v\nwanted: %#v", got, expected)
		return false
	}
	return true
}

func loadPrivateTestData(tkr *tracker.Tracker) {
	users := []string{
		"vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv1",
		"vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv2",
		"vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv3",
	}

	for i, passkey := range users {
		tkr.Store.PutUser(&models.User{
			ID:      uint64(i + 1),
			Passkey: passkey,
		})
	}

	tkr.Store.PutClient("TR2820")

	torrent := &models.Torrent{
		ID:       1,
		Infohash: infoHash,
		Seeders:  models.NewPeerMap(true, tkr.Config),
		Leechers: models.NewPeerMap(false, tkr.Config),
	}

	tkr.Store.PutTorrent(torrent)
}
