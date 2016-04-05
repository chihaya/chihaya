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

func (d *peerStoreDriver) New(storecfg *store.DriverConfig) (store.PeerStore, error) {
	cfg, err := newPeerStoreConfig(storecfg)
	if err != nil {
		return nil, err
	}

	shards := make([]*peerShard, cfg.Shards)
	for i := 0; i < cfg.Shards; i++ {
		shards[i] = &peerShard{}
		shards[i].peers = make(map[string]map[string]peer)
	}
	return &peerStore{
		shards: shards,
	}, nil
}

type peerStoreConfig struct {
	Shards int `yaml:"shards"`
}

func newPeerStoreConfig(storecfg *store.DriverConfig) (*peerStoreConfig, error) {
	bytes, err := yaml.Marshal(storecfg.Config)
	if err != nil {
		return nil, err
	}

	var cfg peerStoreConfig
	err = yaml.Unmarshal(bytes, &cfg)
	if err != nil {
		return nil, err
	}

	if cfg.Shards < 1 {
		cfg.Shards = 1
	}
	return &cfg, nil
}

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

func (s *peerStore) shardIndex(infoHash chihaya.InfoHash) uint32 {
	idx := fnv.New32()
	idx.Write([]byte(infoHash))
	return idx.Sum32() % uint32(len(s.shards))
}

func peerKey(p chihaya.Peer) string {
	return string(p.IP) + string(p.ID)
}

func seedsKey(infoHash chihaya.InfoHash) string {
	return string(infoHash) + "-s"
}

func leechesKey(infoHash chihaya.InfoHash) string {
	return string(infoHash) + "-l"
}

func (s *peerStore) PutSeeder(infoHash chihaya.InfoHash, p chihaya.Peer) error {
	key := seedsKey(infoHash)
	shard := s.shards[s.shardIndex(infoHash)]
	shard.Lock()
	defer shard.Unlock()

	if shard.peers[key] == nil {
		shard.peers[key] = make(map[string]peer)
	}

	shard.peers[key][peerKey(p)] = peer{
		Peer:       p,
		LastAction: time.Now(),
	}

	return nil
}

func (s *peerStore) DeleteSeeder(infoHash chihaya.InfoHash, p chihaya.Peer) error {
	key := seedsKey(infoHash)
	shard := s.shards[s.shardIndex(infoHash)]
	shard.Lock()
	defer shard.Unlock()

	if shard.peers[key] == nil {
		return nil
	}

	delete(shard.peers[key], peerKey(p))

	if len(shard.peers[key]) == 0 {
		delete(shard.peers, key)
	}

	return nil
}

func (s *peerStore) PutLeecher(infoHash chihaya.InfoHash, p chihaya.Peer) error {
	key := leechesKey(infoHash)
	shard := s.shards[s.shardIndex(infoHash)]
	shard.Lock()
	defer shard.Unlock()

	if shard.peers[key] == nil {
		shard.peers[key] = make(map[string]peer)
	}

	shard.peers[key][peerKey(p)] = peer{
		Peer:       p,
		LastAction: time.Now(),
	}

	return nil
}

func (s *peerStore) DeleteLeecher(infoHash chihaya.InfoHash, p chihaya.Peer) error {
	key := leechesKey(infoHash)
	shard := s.shards[s.shardIndex(infoHash)]
	shard.Lock()
	defer shard.Unlock()

	if shard.peers[key] == nil {
		return nil
	}

	delete(shard.peers[key], peerKey(p))

	if len(shard.peers[key]) == 0 {
		delete(shard.peers, key)
	}

	return nil
}

func (s *peerStore) GraduateLeecher(infoHash chihaya.InfoHash, p chihaya.Peer) error {
	lkey := leechesKey(infoHash)
	skey := seedsKey(infoHash)
	shard := s.shards[s.shardIndex(infoHash)]
	shard.Lock()
	defer shard.Unlock()

	if shard.peers[lkey] != nil {
		delete(shard.peers[lkey], peerKey(p))
	}

	if shard.peers[skey] == nil {
		shard.peers[skey] = make(map[string]peer)
	}

	shard.peers[skey][peerKey(p)] = peer{
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

			for peerKey, p := range shard.peers[key] {
				if p.LastAction.Before(cutoff) {
					delete(shard.peers[key], peerKey)
				}
			}

			if len(shard.peers[key]) == 0 {
				delete(shard.peers, key)
			}

			shard.Unlock()
			runtime.Gosched()
		}

		runtime.Gosched()
	}

	return nil
}

func (s *peerStore) AnnouncePeers(infoHash chihaya.InfoHash, seed bool, numWant int) (peers, peers6 []chihaya.Peer, err error) {
	lkey := leechesKey(infoHash)
	skey := seedsKey(infoHash)
	shard := s.shards[s.shardIndex(infoHash)]
	shard.RLock()
	defer shard.RUnlock()

	if seed {
		// Append leeches as possible.
		leeches := shard.peers[lkey]
		for _, p := range leeches {
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
		// Append as many seeds as possible.
		seeds := shard.peers[skey]
		for _, p := range seeds {
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

		// Append leeches until we reach numWant.
		leeches := shard.peers[lkey]
		if numWant > 0 {
			for _, p := range leeches {
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

func (s *peerStore) GetSeeders(infoHash chihaya.InfoHash) (peers, peers6 []chihaya.Peer, err error) {
	key := seedsKey(infoHash)
	shard := s.shards[s.shardIndex(infoHash)]
	shard.RLock()
	defer shard.RUnlock()

	seeds := shard.peers[key]
	for _, p := range seeds {
		if p.IP.To4() == nil {
			peers6 = append(peers6, p.Peer)
		} else {
			peers = append(peers, p.Peer)
		}
	}
	return
}

func (s *peerStore) GetLeechers(infoHash chihaya.InfoHash) (peers, peers6 []chihaya.Peer, err error) {
	key := leechesKey(infoHash)
	shard := s.shards[s.shardIndex(infoHash)]
	shard.RLock()
	defer shard.RUnlock()

	leeches := shard.peers[key]
	for _, p := range leeches {
		if p.IP.To4() == nil {
			peers6 = append(peers6, p.Peer)
		} else {
			peers = append(peers, p.Peer)
		}
	}
	return
}

func (s *peerStore) NumSeeders(infoHash chihaya.InfoHash) int {
	key := seedsKey(infoHash)
	shard := s.shards[s.shardIndex(infoHash)]
	shard.RLock()
	defer shard.RUnlock()

	return len(shard.peers[key])
}

func (s *peerStore) NumLeechers(infoHash chihaya.InfoHash) int {
	key := leechesKey(infoHash)
	shard := s.shards[s.shardIndex(infoHash)]
	shard.RLock()
	defer shard.RUnlock()

	return len(shard.peers[key])
}
