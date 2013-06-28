// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package storage

type Peer struct {
	ID        string
	UserID    uint64
	TorrentID uint64

	Port uint
	IP   string
	Addr []byte

	Uploaded   uint64
	Downloaded uint64
	Left       uint64
	Seeding    bool

	StartTimeUnix int64
	LastAnnounce  int64
}

type Torrent struct {
	ID             uint64
	Infohash       string
	UpMultiplier   float64
	DownMultiplier float64

	Seeders  map[string]*Peer
	Leechers map[string]*Peer

	Snatched   uint
	Status     int64
	LastAction int64
}

type User struct {
	ID             uint64
	Passkey        string
	UpMultiplier   float64
	DownMultiplier float64
	Slots          int64
	UsedSlots      int64

	SlotsLastChecked int64
}
