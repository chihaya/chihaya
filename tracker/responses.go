// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package tracker

import (
	"time"

	"github.com/chihaya/chihaya/tracker/models"
)

type PeerList []models.Peer

type AnnounceResponse struct {
	Complete, Incomplete  int
	Interval, MinInterval time.Duration
	IPv4Peers, IPv6Peers  PeerList

	Compact bool
}

type ScrapeResponse struct {
	Files []*models.Torrent
}

type Writer interface {
	WriteError(error) error
	WriteAnnounce(*AnnounceResponse) error
	WriteScrape(*ScrapeResponse) error
}
