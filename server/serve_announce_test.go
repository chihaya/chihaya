// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package server

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/drivers/backend"
	_ "github.com/chihaya/chihaya/drivers/backend/mock"
	"github.com/chihaya/chihaya/drivers/tracker"
	_ "github.com/chihaya/chihaya/drivers/tracker/mock"
	"github.com/chihaya/chihaya/models"
)

func TestAnnounce(t *testing.T) {
	s, err := New(config.New())
	if err != nil {
		t.Error(err)
	}

	err = s.Prime(func(t tracker.Pool, b backend.Conn) (err error) {
		conn, err := t.Get()
		if err != nil {
			return
		}

		err = conn.AddUser(&models.User{
			ID:      1,
			Passkey: "yby47f04riwpndba456rqxtmifenq5h6",
		})
		if err != nil {
			return
		}

		err = conn.WhitelistClient("TR2820")
		if err != nil {
			return
		}

		torrent := &models.Torrent{
			ID:       1,
			Infohash: string([]byte{0x89, 0xd4, 0xbc, 0x52, 0x11, 0x16, 0xca, 0x1d, 0x42, 0xa2, 0xf3, 0x0d, 0x1f, 0x27, 0x4d, 0x94, 0xe4, 0x68, 0x1d, 0xaf}),
			Seeders:  make(map[string]models.Peer),
			Leechers: make(map[string]models.Peer),
		}

		err = conn.AddTorrent(torrent)
		if err != nil {
			return
		}

		err = conn.AddLeecher(torrent, &models.Peer{
			ID:        "-TR2820-l71jtqkl898b",
			UserID:    1,
			TorrentID: torrent.ID,
			IP:        net.ParseIP("127.0.0.1"),
			Port:      34000,
			Left:      0,
		})

		return
	})
	if err != nil {
		t.Error(err)
	}

	url := "http://localhost:6881/yby47f04riwpndba456rqxtmifenq5h6/announce?info_hash=%89%d4%bcR%11%16%ca%1dB%a2%f3%0d%1f%27M%94%e4h%1d%af&peer_id=-TR2820-l71jtqkl898b&port=51413&uploaded=0&downloaded=0&left=0&numwant=1&key=3c8e3319&compact=0&supportcrypto=1"
	r, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Error(err)
	}

	w := httptest.NewRecorder()
	s.serveAnnounce(w, r)

	if w.Body.String() != "d8:completei1e10:incompletei1e8:intervali1800e12:min intervali900e5:peersld2:ip9:127.0.0.17:peer id20:-TR2820-l71jtqkl898b4:porti34000eeee" {
		t.Errorf("improper response from server:\n%s", w.Body.String())
	}

}
