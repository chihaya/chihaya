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
	"github.com/stretchr/testify/require"
)

func peerInSlice(peer chihaya.Peer, peers []chihaya.Peer) bool {
	for _, v := range peers {
		if v.Equal(peer) {
			return true
		}
	}
	return false
}

func TestPeerStoreAPI(t *testing.T) {
	var (
		hash = chihaya.InfoHash([20]byte{})

		peers = []struct {
			seeder bool
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
			Name:   "memory",
			Config: unmarshalledConfig,
		}
		d = &peerStoreDriver{}
	)
	s, err := d.New(&config)
	require.Nil(t, err)
	require.NotNil(t, s)

	for _, p := range peers {
		// Construct chihaya.Peer from test data.
		peer := chihaya.Peer{
			ID:   chihaya.PeerIDFromString(p.peerID),
			IP:   net.ParseIP(p.ip),
			Port: p.port,
		}

		if p.seeder {
			err = s.PutSeeder(hash, peer)
		} else {
			err = s.PutLeecher(hash, peer)
		}
		require.Nil(t, err)
	}

	leechers1, leechers61, err := s.GetLeechers(hash)
	require.Nil(t, err)
	require.NotEmpty(t, leechers1)
	require.NotEmpty(t, leechers61)
	num := s.NumLeechers(hash)
	require.Equal(t, len(leechers1)+len(leechers61), num)

	seeders1, seeders61, err := s.GetSeeders(hash)
	require.Nil(t, err)
	require.NotEmpty(t, seeders1)
	require.NotEmpty(t, seeders61)
	num = s.NumSeeders(hash)
	require.Equal(t, len(seeders1)+len(seeders61), num)

	leechers := append(leechers1, leechers61...)
	seeders := append(seeders1, seeders61...)

	for _, p := range peers {
		// Construct chihaya.Peer from test data.
		peer := chihaya.Peer{
			ID:   chihaya.PeerIDFromString(p.peerID),
			IP:   net.ParseIP(p.ip),
			Port: p.port,
		}

		if p.seeder {
			require.True(t, peerInSlice(peer, seeders))
		} else {
			require.True(t, peerInSlice(peer, leechers))
		}

		if p.seeder {
			err = s.DeleteSeeder(hash, peer)
		} else {
			err = s.DeleteLeecher(hash, peer)
		}
		require.Nil(t, err)
	}

	require.Zero(t, s.NumLeechers(hash))
	require.Zero(t, s.NumSeeders(hash))

	// Re-add all the peers to the peerStore.
	for _, p := range peers {
		// Construct chihaya.Peer from test data.
		peer := chihaya.Peer{
			ID:   chihaya.PeerIDFromString(p.peerID),
			IP:   net.ParseIP(p.ip),
			Port: p.port,
		}
		if p.seeder {
			s.PutSeeder(hash, peer)
		} else {
			s.PutLeecher(hash, peer)
		}
	}

	// Check that there are 6 seeders, and 4 leechers.
	require.Equal(t, 6, s.NumSeeders(hash))
	require.Equal(t, 4, s.NumLeechers(hash))
	peer := chihaya.Peer{
		ID:   chihaya.PeerIDFromString(peers[0].peerID),
		IP:   net.ParseIP(peers[0].ip),
		Port: peers[0].port,
	}
	err = s.GraduateLeecher(hash, peer)
	require.Nil(t, err)
	// Check that there are 7 seeders, and 3 leechers after graduating a
	// leecher to a seeder.
	require.Equal(t, 7, s.NumSeeders(hash))
	require.Equal(t, 3, s.NumLeechers(hash))

	peers1, peers61, err := s.AnnouncePeers(hash, true, 5, peer, chihaya.Peer{})
	require.Nil(t, err)
	require.NotNil(t, peers1)
	require.NotNil(t, peers61)

	err = s.CollectGarbage(time.Now())
	require.Nil(t, err)
	require.Equal(t, s.NumLeechers(hash), 0)
	require.Equal(t, s.NumSeeders(hash), 0)

	errChan := s.Stop()
	err = <-errChan
	require.Nil(t, err, "PeerStore shutdown must not fail")
}
