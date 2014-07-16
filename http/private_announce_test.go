// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package http

import (
	"net"
	"testing"

	"github.com/chihaya/bencode"
	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/models"

	_ "github.com/chihaya/chihaya/drivers/backend/noop"
	_ "github.com/chihaya/chihaya/drivers/tracker/memory"
)

func loadTestData(tkr *Tracker) error {
	conn, err := tkr.tp.Get()
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
		Seeders:  make(map[string]models.Peer),
		Leechers: make(map[string]models.Peer),
	}

	err = conn.PutTorrent(torrent)
	if err != nil {
		return err
	}

	peer := &models.Peer{
		ID:        "-TR2820-peer1",
		UserID:    1,
		TorrentID: torrent.ID,
		IP:        net.ParseIP("127.0.0.1"),
		Port:      1234,
		Left:      0,
	}

	err = conn.PutLeecher(torrent.Infohash, peer)
	if err != nil {
		return err
	}

	peer = &*peer
	peer.ID = "-TR2820-peer3"
	peer.UserID = 3

	return conn.PutLeecher(torrent.Infohash, peer)
}

func TestPrivateAnnounce(t *testing.T) {
	cfg := config.DefaultConfig
	cfg.Private = true

	tkr, err := NewTracker(&cfg)
	if err != nil {
		t.Fatal(err)
	}

	err = loadTestData(tkr)
	if err != nil {
		t.Fatal(err)
	}

	srv, err := createServer(tkr, &cfg)
	if err != nil {
		t.Fatal(err)
	}

	srv.URL = srv.URL + "/users/vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv2"
	defer srv.Close()

	peer := makePeerParams("-TR2820-peer2", true)
	expected := makeResponse(1, 2, bencode.List{
		makePeerResponse("-TR2820-peer1"),
		makePeerResponse("-TR2820-peer3"),
	})
	checkAnnounce(peer, expected, srv, t)
}
