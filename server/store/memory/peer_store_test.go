// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package memory

import (
	"net"
	"testing"
	"time"

	"github.com/chihaya/chihaya"
	"github.com/chihaya/chihaya/server/store"
	"github.com/stretchr/testify/assert"
)

func peerInSlice(peer chihaya.Peer, peers []chihaya.Peer) bool {
	for _, v := range peers {
		if v.Equal(&peer) {
			return true
		}
	}
	return false
}

func TestPeerStoreAPI(t *testing.T) {
	var (
		hash = chihaya.InfoHash("11111111111111111111")

		peers = []struct {
			seed   bool
			peerID string
			ip     string
			port   uint16
		}{
			{false, "-AZ3034-6wfG2wk6wWLc", "250.183.81.177", 5720},
			{false, "-AZ3042-6ozMq5q6Q3NX", "38.241.13.19", 4833},
			{false, "-BS5820-oy4La2MWGEFj", "fd45:7856:3dae::48", 2878},
			{false, "-AR6360-6oZyyMWoOOBe", "fd0a:29a8:8445::38", 3167},
			{true, "-AG2083-s1hiF8vGAAg0", "231.231.49.173", 1453},
			{true, "-AG3003-lEl2Mm4NEO4n", "254.99.84.77", 7032},
			{true, "-MR1100-00HS~T7*65rm", "211.229.100.129", 2614},
			{true, "-LK0140-ATIV~nbEQAMr", "fdad:c435:bf79::12", 4114},
			{true, "-KT2210-347143496631", "fdda:1b35:7d6e::9", 6179},
			{true, "-TR0960-6ep6svaa61r4", "fd7f:78f0:4c77::55", 4727},
		}
		unmarshalledConfig = struct {
			Shards int
		}{
			1,
		}
		config = store.DriverConfig{
			"memory",
			unmarshalledConfig,
		}
		d = &peerStoreDriver{}
	)
	s, err := d.New(&config)
	assert.Nil(t, err)
	assert.NotNil(t, s)

	for _, p := range peers {
		// Construct chihaya.Peer from test data.
		peer := chihaya.Peer{
			chihaya.PeerID(p.peerID),
			net.ParseIP(p.ip),
			p.port,
		}

		if p.seed {
			err = s.PutSeeder(hash, peer)
		} else {
			err = s.PutLeecher(hash, peer)
		}
		assert.Nil(t, err)
	}

	leeches1, leeches61, err := s.GetLeechers(hash)
	assert.Nil(t, err)
	assert.NotEmpty(t, leeches1)
	assert.NotEmpty(t, leeches61)
	num := s.NumLeechers(hash)
	assert.Equal(t, len(leeches1)+len(leeches61), num)

	seeds1, seeds61, err := s.GetSeeders(hash)
	assert.Nil(t, err)
	assert.NotEmpty(t, seeds1)
	assert.NotEmpty(t, seeds61)
	num = s.NumSeeders(hash)
	assert.Equal(t, len(seeds1)+len(seeds61), num)

	leeches := append(leeches1, leeches61...)
	seeds := append(seeds1, seeds61...)

	for _, p := range peers {
		// Construct chihaya.Peer from test data.
		peer := chihaya.Peer{
			chihaya.PeerID(p.peerID),
			net.ParseIP(p.ip),
			p.port,
		}

		if p.seed {
			assert.True(t, peerInSlice(peer, seeds))
		} else {
			assert.True(t, peerInSlice(peer, leeches))
		}

		if p.seed {
			err = s.DeleteSeeder(hash, peer)
		} else {
			err = s.DeleteLeecher(hash, peer)
		}
		assert.Nil(t, err)
	}

	assert.Zero(t, s.NumLeechers(hash))
	assert.Zero(t, s.NumSeeders(hash))

	// Re-add all the peers to the peerStore.
	for _, p := range peers {
		// Construct chihaya.Peer from test data.
		peer := chihaya.Peer{
			chihaya.PeerID(p.peerID),
			net.ParseIP(p.ip),
			p.port,
		}
		if p.seed {
			s.PutSeeder(hash, peer)
		} else {
			s.PutLeecher(hash, peer)
		}
	}

	// Check that there are 6 seeds, and 4 leeches.
	assert.Equal(t, 6, s.NumSeeders(hash))
	assert.Equal(t, 4, s.NumLeechers(hash))
	peer := chihaya.Peer{
		chihaya.PeerID(peers[0].peerID),
		net.ParseIP(peers[0].ip),
		peers[0].port,
	}
	err = s.GraduateLeecher(hash, peer)
	assert.Nil(t, err)
	// Check that there are 7 seeds, and 3 leeches after graduating a
	// leech to a seed.
	assert.Equal(t, 7, s.NumSeeders(hash))
	assert.Equal(t, 3, s.NumLeechers(hash))

	peers1, peers61, err := s.AnnouncePeers(hash, true, 5)
	assert.Nil(t, err)
	assert.NotNil(t, peers1)
	assert.NotNil(t, peers61)

	err = s.CollectGarbage(time.Now())
	assert.Nil(t, err)
	assert.Equal(t, s.NumLeechers(hash), 0)
	assert.Equal(t, s.NumSeeders(hash), 0)
}
