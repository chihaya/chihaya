// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package storage implements a high-level abstraction over the multiple
// data stores used by a BitTorrent tracker.
package storage

import (
	"strconv"
)

type Peer struct {
	ID        string `json:"id"`
	UserID    uint64 `json:"user_id"`
	TorrentID uint64 `json:"torrent_id"`

	IP   string `json:"ip"`
	Port uint64 `json:"port"`

	Uploaded     uint64 `json:"uploaded"`
	Downloaded   uint64 `json:"downloaded`
	Left         uint64 `json:"left"`
	LastAnnounce int64  `json:"last_announce"`
}

func PeerMapKey(peer *Peer) string {
	return peer.ID + ":" + strconv.FormatUint(peer.UserID, 36)
}

type Torrent struct {
	ID       uint64 `json:"id"`
	Infohash string `json:"infohash"`
	Active   bool   `json:"active"`

	Seeders  map[string]Peer `json:"seeders"`
	Leechers map[string]Peer `json:"leechers"`

	Snatches       uint64  `json:"snatches"`
	UpMultiplier   float64 `json:"up_multiplier"`
	DownMultiplier float64 `json:"down_multiplier"`
	LastAction     int64   `json:"last_action"`
}

type User struct {
	ID      uint64 `json:"id"`
	Passkey string `json:"passkey"`

	UpMultiplier   float64 `json:"up_multiplier"`
	DownMultiplier float64 `json:"down_multiplier"`
	Slots          int64   `json:"slots"`
	SlotsUsed      int64   `json:"slots_used"`
	Snatches       uint64  `json:"snatches"`
}
