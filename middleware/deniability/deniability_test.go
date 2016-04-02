// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package deniability

import (
	"fmt"
	"math/rand"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/chihaya/chihaya"
)

type constructorTestData struct {
	cfg   Config
	error bool
}

var constructorData = []constructorTestData{
	{Config{1.0, 10, "abc", 1024, 1025}, false},
	{Config{1.1, 10, "abc", 1024, 1025}, true},
	{Config{0, 10, "abc", 1024, 1025}, true},
	{Config{1.0, 0, "abc", 1024, 1025}, true},
	{Config{1.0, 10, "01234567890123456789_", 1024, 1025}, true},
	{Config{1.0, 10, "abc", 0, 1025}, true},
	{Config{1.0, 10, "abc", 1024, 0}, true},
	{Config{1.0, 10, "abc", 1024, 65537}, true},
}

func TestReplacePeer(t *testing.T) {
	cfg := Config{
		Prefix:  "abc",
		MinPort: 1024,
		MaxPort: 1025,
	}
	mw := deniabilityMiddleware{
		r:   rand.New(rand.NewSource(0)),
		cfg: &cfg,
	}
	peer := chihaya.Peer{
		ID:   chihaya.PeerID("abcdefghijklmnoprstu"),
		Port: 2000,
		IP:   net.ParseIP("10.150.255.23"),
	}
	peers := []chihaya.Peer{peer}

	mw.replacePeer(peers, false)
	assert.Equal(t, 1, len(peers))
	assert.Equal(t, "abc", string(peers[0].ID[:3]))
	assert.Equal(t, uint16(1024), peers[0].Port)
	assert.NotNil(t, peers[0].IP.To4())

	mw.replacePeer(peers, true)
	assert.Equal(t, 1, len(peers))
	assert.Equal(t, "abc", string(peers[0].ID[:3]))
	assert.Equal(t, uint16(1024), peers[0].Port)
	assert.Nil(t, peers[0].IP.To4())

	peers = []chihaya.Peer{peer, peer}

	mw.replacePeer(peers, true)
	assert.True(t, (peers[0].Port == peer.Port) != (peers[1].Port == peer.Port), "not exactly one peer was replaced")
}

func TestInsertPeer(t *testing.T) {
	cfg := Config{
		Prefix:  "abc",
		MinPort: 1024,
		MaxPort: 1025,
	}
	mw := deniabilityMiddleware{
		r:   rand.New(rand.NewSource(0)),
		cfg: &cfg,
	}
	peer := chihaya.Peer{
		ID:   chihaya.PeerID("abcdefghijklmnoprstu"),
		Port: 2000,
		IP:   net.ParseIP("10.150.255.23"),
	}
	var peers []chihaya.Peer

	peers = mw.insertPeer(peers, false)
	assert.Equal(t, 1, len(peers))
	assert.Equal(t, uint16(1024), peers[0].Port)
	assert.Equal(t, "abc", string(peers[0].ID[:3]))
	assert.NotNil(t, peers[0].IP.To4())

	peers = []chihaya.Peer{peer, peer}

	peers = mw.insertPeer(peers, true)
	assert.Equal(t, 3, len(peers))
}

func TestConstructor(t *testing.T) {
	for _, tt := range constructorData {
		_, err := constructor(chihaya.MiddlewareConfig{
			Config: tt.cfg,
		})

		if tt.error {
			assert.NotNil(t, err, fmt.Sprintf("error expected for %+v", tt.cfg))
		} else {
			assert.Nil(t, err, fmt.Sprintf("no error expected for %+v", tt.cfg))
		}
	}
}
