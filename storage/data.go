// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package storage

type Peer struct {
	Id        string
	UserId    uint64
	TorrentId uint64

	Port uint
	Ip   string
	Addr []byte

	Uploaded   uint64
	Downloaded uint64
	Left       uint64
	Seeding    bool

	StartTime    int64 // Unix Timestamp
	LastAnnounce int64
}

type Torrent struct {
	Id             uint64
	InfoHash       string
	UpMultiplier   float64
	DownMultiplier float64

	Seeders  map[string]*Peer
	Leechers map[string]*Peer

	Snatched   uint
	Status     int64
	LastAction int64
}

type User struct {
	Id             uint64
	Passkey        string
	UpMultiplier   float64
	DownMultiplier float64
	Slots          int64
	UsedSlots      int64

	SlotsLastChecked int64
}
