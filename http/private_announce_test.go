// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package http

import (
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/chihaya/bencode"
	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/drivers/backend"
	"github.com/chihaya/chihaya/drivers/tracker"
	"github.com/chihaya/chihaya/models"

	_ "github.com/chihaya/chihaya/drivers/backend/noop"
	_ "github.com/chihaya/chihaya/drivers/tracker/memory"
)

type primer func(tracker.Pool, backend.Conn) error

func (t *Tracker) prime(p primer) error {
	return p(t.tp, t.bc)
}

func loadTestData(tkr *Tracker) (err error) {
	return tkr.prime(func(tp tracker.Pool, bc backend.Conn) (err error) {
		conn, err := tp.Get()
		if err != nil {
			return
		}

		err = conn.PutUser(&models.User{
			ID:      1,
			Passkey: "yby47f04riwpndba456rqxtmifenqxx1",
		})
		if err != nil {
			return
		}
		err = conn.PutUser(&models.User{
			ID:      2,
			Passkey: "yby47f04riwpndba456rqxtmifenqxx2",
		})
		if err != nil {
			return
		}
		err = conn.PutUser(&models.User{
			ID:      3,
			Passkey: "yby47f04riwpndba456rqxtmifenqxx3",
		})
		if err != nil {
			return
		}

		err = conn.PutClient("TR2820")
		if err != nil {
			return
		}

		torrent := &models.Torrent{
			ID:       1,
			Infohash: string([]byte{0x89, 0xd4, 0xbc, 0x52, 0x11, 0x16, 0xca, 0x1d, 0x42, 0xa2, 0xf3, 0x0d, 0x1f, 0x27, 0x4d, 0x94, 0xe4, 0x68, 0x1d, 0xaf}),
			Seeders:  make(map[string]models.Peer),
			Leechers: make(map[string]models.Peer),
		}

		err = conn.PutTorrent(torrent)
		if err != nil {
			return
		}

		err = conn.PutLeecher(torrent.Infohash, &models.Peer{
			ID:        "-TR2820-l71jtqkl8xx1",
			UserID:    1,
			TorrentID: torrent.ID,
			IP:        net.ParseIP("127.0.0.1"),
			Port:      34000,
			Left:      0,
		})
		if err != nil {
			return
		}

		err = conn.PutLeecher(torrent.Infohash, &models.Peer{
			ID:        "-TR2820-l71jtqkl8xx3",
			UserID:    3,
			TorrentID: torrent.ID,
			IP:        net.ParseIP("2001::53aa:64c:0:7f83:bc43:dec9"),
			Port:      34000,
			Left:      0,
		})

		return
	})
}

func testRoute(cfg *config.Config, url string) ([]byte, error) {
	tkr, err := NewTracker(cfg)
	if err != nil {
		return nil, err
	}

	err = loadTestData(tkr)
	if err != nil {
		return nil, err
	}

	srv := httptest.NewServer(setupRoutes(tkr, cfg))
	defer srv.Close()

	resp, err := http.Get(srv.URL + url)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}

	return body, nil
}

func TestPrivateAnnounce(t *testing.T) {
	cfg := config.DefaultConfig
	cfg.Private = true

	url := "/users/yby47f04riwpndba456rqxtmifenqxx2/announce?info_hash=%89%d4%bcR%11%16%ca%1dB%a2%f3%0d%1f%27M%94%e4h%1d%af&peer_id=-TR2820-l71jtqkl898b&port=51413&uploaded=0&downloaded=0&left=0&numwant=1&key=3c8e3319&compact=0"

	expected := bencode.Dict{
		"complete":     int64(1),
		"incomplete":   int64(2),
		"interval":     int64(1800),
		"min interval": int64(900),
		"peers": bencode.List{
			bencode.Dict{
				"ip":      "127.0.0.1",
				"peer id": "-TR2820-l71jtqkl8xx1",
				"port":    int64(34000),
			},
		},
	}

	response, err := testRoute(&cfg, url)
	if err != nil {
		t.Error(err)
	}
	got, err := bencode.Unmarshal(response)

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("\ngot:    %#v\nwanted: %#v", got, expected)
	}

	url = "/users/yby47f04riwpndba456rqxtmifenqxx2/announce?info_hash=%89%d4%bcR%11%16%ca%1dB%a2%f3%0d%1f%27M%94%e4h%1d%af&peer_id=-TR2820-l71jtqkl898b&port=51413&uploaded=0&downloaded=0&left=0&numwant=2&key=3c8e3319&compact=0"

	expected = bencode.Dict{
		"complete":     int64(1),
		"incomplete":   int64(2),
		"interval":     int64(1800),
		"min interval": int64(900),
		"peers": bencode.List{
			bencode.Dict{
				"ip":      "127.0.0.1",
				"peer id": "-TR2820-l71jtqkl8xx1",
				"port":    int64(34000),
			},
			bencode.Dict{
				"ip":      "2001:0:53aa:64c:0:7f83:bc43:dec9",
				"peer id": "-TR2820-l71jtqkl8xx3",
				"port":    int64(34000),
			},
		},
	}

	response, err = testRoute(&cfg, url)
	if err != nil {
		t.Error(err)
	}
	got, err = bencode.Unmarshal(response)

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("\ngot:    %#v\nwanted: %#v", got, expected)
	}
}
