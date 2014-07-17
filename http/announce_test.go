// Copyright 2014 The Chihaya Authors. All rights reserved.
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

	// Add one seeder.
	peer := makePeerParams("peer1", true)
	expected := makeResponse(1, 0, bencode.List{})
	checkAnnounce(peer, expected, srv, t)

	// Add another seeder.
	peer = makePeerParams("peer2", true)
	expected = makeResponse(2, 0, bencode.List{})
	checkAnnounce(peer, expected, srv, t)

	// Add a leecher.
	peer = makePeerParams("peer3", false)
	expected = makeResponse(2, 1, bencode.List{
		makePeerResponse("peer1"),
		makePeerResponse("peer2"),
	})
	checkAnnounce(peer, expected, srv, t)

	// Remove seeder.
	peer = makePeerParams("peer1", true)
	peer["event"] = "stopped"
	expected = makeResponse(1, 1, nil)
	checkAnnounce(peer, expected, srv, t)

	// Check seeders.
	peer = makePeerParams("peer3", false)
	expected = makeResponse(1, 1, bencode.List{
		makePeerResponse("peer2"),
	})
	checkAnnounce(peer, expected, srv, t)
}

func TestTorrentPurging(t *testing.T) {
	cfg := config.DefaultConfig
	srv, err := setupTracker(&cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	torrentApiPath := srv.URL + "/torrents/" + url.QueryEscape(infoHash)

	// Add one seeder.
	peer := makePeerParams("peer1", true)
	announce(peer, srv)

	_, status, err := fetchPath(torrentApiPath)
	if err != nil {
		t.Fatal(err)
	} else if status != http.StatusOK {
		t.Fatalf("expected torrent to exist (got %s)", http.StatusText(status))
	}

	// Remove seeder.
	peer = makePeerParams("peer1", true)
	peer["event"] = "stopped"
	announce(peer, srv)

	_, status, err = fetchPath(torrentApiPath)
	if err != nil {
		t.Fatal(err)
	} else if status != http.StatusNotFound {
		t.Fatalf("expected torrent to have been purged (got %s)", http.StatusText(status))
	}
}

func TestStalePeerPurging(t *testing.T) {
	cfg := config.DefaultConfig
	cfg.Announce = config.Duration{10 * time.Millisecond}

	srv, err := setupTracker(&cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	torrentApiPath := srv.URL + "/torrents/" + url.QueryEscape(infoHash)

	// Add one seeder.
	peer := makePeerParams("peer1", true)
	announce(peer, srv)

	_, status, err := fetchPath(torrentApiPath)
	if err != nil {
		t.Fatal(err)
	} else if status != http.StatusOK {
		t.Fatalf("expected torrent to exist (got %s)", http.StatusText(status))
	}

	// Add a leecher.
	peer = makePeerParams("peer2", false)
	expected := makeResponse(1, 1, bencode.List{
		makePeerResponse("peer1"),
	})
	expected["interval"] = int64(0)
	checkAnnounce(peer, expected, srv, t)

	// Let them both expire.
	time.Sleep(30 * time.Millisecond)

	_, status, err = fetchPath(torrentApiPath)
	if err != nil {
		t.Fatal(err)
	} else if status != http.StatusNotFound {
		t.Fatalf("expected torrent to have been purged (got %s)", http.StatusText(status))
	}
}

func TestPrivateAnnounce(t *testing.T) {
	cfg := config.DefaultConfig
	cfg.Private = true

	tkr, err := tracker.New(&cfg)
	if err != nil {
		t.Fatal(err)
	}

	err = loadPrivateTestData(tkr)
	if err != nil {
		t.Fatal(err)
	}

	srv, err := createServer(tkr, &cfg)
	if err != nil {
		t.Fatal(err)
	}

	defer srv.Close()
	baseURL := srv.URL

	peer := makePeerParams("-TR2820-peer1", false)
	expected := makeResponse(0, 1, bencode.List{})
	srv.URL = baseURL + "/users/vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv1"
	checkAnnounce(peer, expected, srv, t)

	peer = makePeerParams("-TR2820-peer2", false)
	expected = makeResponse(0, 2, bencode.List{
		makePeerResponse("-TR2820-peer1"),
	})
	srv.URL = baseURL + "/users/vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv2"
	checkAnnounce(peer, expected, srv, t)

	peer = makePeerParams("-TR2820-peer3", true)
	expected = makeResponse(1, 2, bencode.List{
		makePeerResponse("-TR2820-peer1"),
		makePeerResponse("-TR2820-peer2"),
	})
	srv.URL = baseURL + "/users/vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv3"
	checkAnnounce(peer, expected, srv, t)

	peer = makePeerParams("-TR2820-peer1", false)
	expected = makeResponse(1, 2, bencode.List{
		makePeerResponse("-TR2820-peer2"),
		makePeerResponse("-TR2820-peer3"),
	})
	srv.URL = baseURL + "/users/vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv1"
	checkAnnounce(peer, expected, srv, t)
}

func TestPreferredSubnet(t *testing.T) {
	cfg := config.DefaultConfig
	cfg.PreferredSubnet = true
	cfg.PreferredIPv4Subnet = 8
	cfg.PreferredIPv6Subnet = 8

	srv, err := setupTracker(&cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	// Make a bunch of peers in two subnets.
	peerA1 := makePeerParams("peerA1", false)
	peerA1["ip"] = "44.0.0.1"

	peerA2 := makePeerParams("peerA2", false)
	peerA2["ip"] = "44.0.0.2"

	peerA3 := makePeerParams("peerA3", false)
	peerA3["ip"] = "44.0.0.3"

	peerA4 := makePeerParams("peerA4", false)
	peerA4["ip"] = "44.0.0.4"

	peerB1 := makePeerParams("peerB1", false)
	peerB1["ip"] = "45.0.0.1"

	peerB2 := makePeerParams("peerB2", false)
	peerB2["ip"] = "45.0.0.2"

	// Check what peers their announces return.
	expected := makeResponse(0, 1, bencode.List{})
	checkAnnounce(peerA1, expected, srv, t)

	expected = makeResponse(0, 2, bencode.List{
		peerFromParams(peerA1),
	})
	checkAnnounce(peerA2, expected, srv, t)

	expected = makeResponse(0, 3, bencode.List{
		peerFromParams(peerA1),
		peerFromParams(peerA2),
	})
	checkAnnounce(peerB1, expected, srv, t)

	peerB2["numwant"] = "1"
	expected = makeResponse(0, 4, bencode.List{
		peerFromParams(peerB1),
	})
	checkAnnounce(peerB2, expected, srv, t)
	checkAnnounce(peerB2, expected, srv, t)
	checkAnnounce(peerB2, expected, srv, t)

	peerA3["numwant"] = "2"
	expected = makeResponse(0, 5, bencode.List{
		peerFromParams(peerA1),
		peerFromParams(peerA2),
	})
	checkAnnounce(peerA3, expected, srv, t)

	peerA4["numwant"] = "3"
	expected = makeResponse(0, 6, bencode.List{
		peerFromParams(peerA1),
		peerFromParams(peerA2),
		peerFromParams(peerA3),
	})
	checkAnnounce(peerA4, expected, srv, t)
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
		"peer id": id,
		"ip":      "127.0.0.1",
		"port":    int64(1234),
	}
}

func peerFromParams(peer params) bencode.Dict {
	ip := peer["ip"]
	if ip == "" {
		ip = "127.0.0.1"
	}

	port, _ := strconv.ParseInt(peer["port"], 10, 64)

	return bencode.Dict{
		"peer id": peer["peer_id"],
		"ip":      ip,
		"port":    port,
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

func loadPrivateTestData(tkr *tracker.Tracker) error {
	conn, err := tkr.Pool.Get()
	if err != nil {
		return err
	}

	users := []string{
		"vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv1",
		"vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv2",
		"vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv3",
	}

	for i, passkey := range users {
		err = conn.PutUser(&models.User{
			ID:      uint64(i + 1),
			Passkey: passkey,
		})

		if err != nil {
			return err
		}
	}

	err = conn.PutClient("TR2820")
	if err != nil {
		return err
	}

	torrent := &models.Torrent{
		ID:       1,
		Infohash: infoHash,
		Seeders:  models.PeerMap{},
		Leechers: models.PeerMap{},
	}

	return conn.PutTorrent(torrent)
}
