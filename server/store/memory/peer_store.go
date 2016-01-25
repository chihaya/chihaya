// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package memory

import (
	"hash/fnv"
	"runtime"
	"sync"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/chihaya/chihaya"
	"github.com/chihaya/chihaya/server/store"
)

func init() {
	store.RegisterPeerStoreDriver("memory", &peerStoreDriver{})
}

type peerStoreDriver struct{}

func (d *peerStoreDriver) New(storecfg *store.Config) (store.PeerStore, error) {
	cfg, err := newPeerStoreConfig(storecfg)
	if err != nil {
		return nil, err
	}

	return &peerStore{
		shards: make([]*peerShard, cfg.Shards),
	}, nil
}

type peerStoreConfig struct {
	Shards int `yaml:"shards"`
}

func newPeerStoreConfig(storecfg *store.Config) (*peerStoreConfig, error) {
	bytes, err := yaml.Marshal(storecfg.PeerStoreConfig)
	if err != nil {
		return nil, err
	}

	var cfg peerStoreConfig
	err = yaml.Unmarshal(bytes, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

const seedersSuffix = "-seeders"
const leechersSuffix = "-leechers"

type peer struct {
	chihaya.Peer
	LastAction time.Time
}

type peerShard struct {
	peers map[string]map[string]peer
	sync.RWMutex
}

type peerStore struct {
	shards []*peerShard
}

var _ store.PeerStore = &peerStore{}

func (s *peerStore) shardIndex(infohash string) uint32 {
	idx := fnv.New32()
	idx.Write([]byte(infohash))
	return idx.Sum32() % uint32(len(s.shards))
}

func (s *peerStore) PutSeeder(infohash string, p chihaya.Peer) error {
	key := infohash + seedersSuffix

	shard := s.shards[s.shardIndex(infohash)]
	shard.Lock()
	defer shard.Unlock()

	if shard.peers[key] == nil {
		shard.peers[key] = make(map[string]peer)
	}

	shard.peers[key][p.ID] = peer{
		Peer:       p,
		LastAction: time.Now(),
	}

	return nil
}

func (s *peerStore) DeleteSeeder(infohash, peerID string) error {
	key := infohash + seedersSuffix

	shard := s.shards[s.shardIndex(infohash)]
	shard.Lock()
	defer shard.Unlock()

	if shard.peers[key] == nil {
		return nil
	}

	delete(shard.peers[key], peerID)

	if len(shard.peers[key]) == 0 {
		shard.peers[key] = nil
	}

	return nil
}

func (s *peerStore) PutLeecher(infohash string, p chihaya.Peer) error {
	key := infohash + leechersSuffix

	shard := s.shards[s.shardIndex(infohash)]
	shard.Lock()
	defer shard.Unlock()

	if shard.peers[key] == nil {
		shard.peers[key] = make(map[string]peer)
	}

	shard.peers[key][p.ID] = peer{
		Peer:       p,
		LastAction: time.Now(),
	}

	return nil
}

func (s *peerStore) DeleteLeecher(infohash, peerID string) error {
	key := infohash + leechersSuffix

	shard := s.shards[s.shardIndex(infohash)]
	shard.Lock()
	defer shard.Unlock()

	if shard.peers[key] == nil {
		return nil
	}

	delete(shard.peers[key], peerID)

	if len(shard.peers[key]) == 0 {
		shard.peers[key] = nil
	}

	return nil
}

func (s *peerStore) GraduateLeecher(infohash string, p chihaya.Peer) error {
	leecherKey := infohash + leechersSuffix
	seederKey := infohash + seedersSuffix

	shard := s.shards[s.shardIndex(infohash)]
	shard.Lock()
	defer shard.Unlock()

	if shard.peers[leecherKey] != nil {
		delete(shard.peers[leecherKey], p.ID)
	}

	if shard.peers[seederKey] == nil {
		shard.peers[seederKey] = make(map[string]peer)
	}

	shard.peers[seederKey][p.ID] = peer{
		Peer:       p,
		LastAction: time.Now(),
	}

	return nil
}

func (s *peerStore) CollectGarbage(cutoff time.Time) error {
	for _, shard := range s.shards {
		shard.RLock()
		var keys []string
		for key := range shard.peers {
			keys = append(keys, key)
		}
		shard.RUnlock()
		runtime.Gosched()

		for _, key := range keys {
			shard.Lock()
			var peersToDelete []string
			for peerID, p := range shard.peers[key] {
				if p.LastAction.Before(cutoff) {
					peersToDelete = append(peersToDelete, peerID)
				}
			}

			for _, peerID := range peersToDelete {
				delete(shard.peers[key], peerID)
			}
			shard.Unlock()
			runtime.Gosched()
		}

		runtime.Gosched()
	}

	return nil
}

func (s *peerStore) AnnouncePeers(infohash string, seeder bool, numWant int) (peers, peers6 []chihaya.Peer, err error) {
	leecherKey := infohash + leechersSuffix
	seederKey := infohash + seedersSuffix

	shard := s.shards[s.shardIndex(infohash)]
	shard.RLock()
	defer shard.RUnlock()

	if seeder {
		// Append leechers as possible.
		leechers := shard.peers[leecherKey]
		for _, p := range leechers {
			if numWant == 0 {
				break
			}

			if p.IP.To4() == nil {
				peers6 = append(peers6, p.Peer)
			} else {
				peers = append(peers, p.Peer)
			}
			numWant--
		}
	} else {
		// Append as many seeders as possible.
		seeders := shard.peers[seederKey]
		for _, p := range seeders {
			if numWant == 0 {
				break
			}

			if p.IP.To4() == nil {
				peers6 = append(peers6, p.Peer)
			} else {
				peers = append(peers, p.Peer)
			}
			numWant--
		}

		// Append leechers until we reach numWant.
		leechers := shard.peers[leecherKey]
		if numWant > 0 {
			for _, p := range leechers {
				if numWant == 0 {
					break
				}

				if p.IP.To4() == nil {
					peers6 = append(peers6, p.Peer)
				} else {
					peers = append(peers, p.Peer)
				}
				numWant--
			}
		}
	}

	return
}
