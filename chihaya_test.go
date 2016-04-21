// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package chihaya

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	peers = []struct {
		peerID string
		ip     string
		port   uint16
	}{
		{"-AZ3034-6wfG2wk6wWLc", "250.183.81.177", 5720},
		{"-BS5820-oy4La2MWGEFj", "fd45:7856:3dae::48", 2878},
		{"-TR0960-6ep6svaa61r4", "fd45:7856:3dae::48", 2878},
		{"-BS5820-oy4La2MWGEFj", "fd0a:29a8:8445::38", 2878},
		{"-BS5820-oy4La2MWGEFj", "fd45:7856:3dae::48", 8999},
	}
)

func TestPeerEquality(t *testing.T) {
	// Build peers from test data.
	var builtPeers []Peer
	for _, peer := range peers {
		builtPeers = append(builtPeers, Peer{
			ID:   PeerIDFromString(peer.peerID),
			IP:   net.ParseIP(peer.ip),
			Port: peer.port,
		})
	}

	assert.True(t, builtPeers[0].Equal(builtPeers[0]))
	assert.False(t, builtPeers[0].Equal(builtPeers[1]))
	assert.True(t, builtPeers[1].Equal(builtPeers[1]))
	assert.False(t, builtPeers[1].Equal(builtPeers[2]))
	assert.False(t, builtPeers[1].Equal(builtPeers[3]))
	assert.False(t, builtPeers[1].Equal(builtPeers[4]))
}
