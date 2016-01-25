// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package clientid implements the parsing of BitTorrent ClientIDs from
// BitTorrent PeerIDs.
package clientid

// New returns the part of a PeerID that identifies a peer's client software.
func New(peerID string) (clientID string) {
	length := len(peerID)
	if length >= 6 {
		if peerID[0] == '-' {
			if length >= 7 {
				clientID = peerID[1:7]
			}
		} else {
			clientID = peerID[:6]
		}
	}

	return
}
