// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.package middleware

package chihaya

import (
	"net"
	"time"
)

// AnnounceRequest represents the parsed parameters from an announce request.
type AnnounceRequest map[string]interface{}

// AnnounceResponse represents the parameters used to create an announce
// response.
type AnnounceResponse struct {
	Compact     bool
	Complete    int32
	Incomplete  int32
	Interval    time.Duration
	MinInterval time.Duration
	IPv4Peers   []Peer
	IPv6Peers   []Peer
}

// ScrapeRequest represents the parsed parameters from a scrape request.
type ScrapeRequest map[string]interface{}

// ScrapeResponse represents the parameters used to create a scrape response.
type ScrapeResponse struct {
	Files map[string]Scrape
}

// Scrape represents the state of a swarm that is returned in a scrape response.
type Scrape struct {
	Complete   int32
	Incomplete int32
}

// Peer represents the connection details of a peer that is returned in an
// announce response.
type Peer struct {
	ID   string
	IP   net.IP
	Port uint16
}
