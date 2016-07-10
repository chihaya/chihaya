// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package memory

import (
	"encoding/binary"
	"log"
	"net"
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
		shards[i].swarms = make(map[chihaya.InfoHash]swarm)
	}
	return &peerStore{
		shards: shards,
		closed: make(chan struct{}),
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

type serializedPeer string

type peerShard struct {
	swarms map[chihaya.InfoHash]swarm
	sync.RWMutex
}

type swarm struct {
	// map serialized peer to mtime
	seeders  map[serializedPeer]int64
	leechers map[serializedPeer]int64
}

type peerStore struct {
	shards []*peerShard
	closed chan struct{}
}

var _ store.PeerStore = &peerStore{}

func (s *peerStore) shardIndex(infoHash chihaya.InfoHash) uint32 {
	return binary.BigEndian.Uint32(infoHash[:4]) % uint32(len(s.shards))
}

func peerKey(p chihaya.Peer) serializedPeer {
	b := make([]byte, 20+2+len(p.IP))
	copy(b[:20], p.ID[:])
	binary.BigEndian.PutUint16(b[20:22], p.Port)
	copy(b[22:], p.IP)

	return serializedPeer(b)
}

func decodePeerKey(pk serializedPeer) chihaya.Peer {
	return chihaya.Peer{
		ID:   chihaya.PeerIDFromString(string(pk[:20])),
		Port: binary.BigEndian.Uint16([]byte(pk[20:22])),
		IP:   net.IP(pk[22:]),
	}
}

func (s *peerStore) PutSeeder(infoHash chihaya.InfoHash, p chihaya.Peer) error {
	select {
	case <-s.closed:
		panic("attempted to interact with stopped store")
	default:
	}

	shard := s.shards[s.shardIndex(infoHash)]
	shard.Lock()
	defer shard.Unlock()

	if _, ok := shard.swarms[infoHash]; !ok {
		shard.swarms[infoHash] = swarm{
			seeders:  make(map[serializedPeer]int64),
			leechers: make(map[serializedPeer]int64),
		}
	}

	shard.swarms[infoHash].seeders[peerKey(p)] = time.Now().UnixNano()

	return nil
}

func (s *peerStore) DeleteSeeder(infoHash chihaya.InfoHash, p chihaya.Peer) error {
	select {
	case <-s.closed:
		panic("attempted to interact with stopped store")
	default:
	}

	shard := s.shards[s.shardIndex(infoHash)]
	pk := peerKey(p)
	shard.Lock()
	defer shard.Unlock()

	if _, ok := shard.swarms[infoHash]; !ok {
		return store.ErrResourceDoesNotExist
	}

	if _, ok := shard.swarms[infoHash].seeders[pk]; !ok {
		return store.ErrResourceDoesNotExist
	}

	delete(shard.swarms[infoHash].seeders, pk)

	if len(shard.swarms[infoHash].seeders)|len(shard.swarms[infoHash].leechers) == 0 {
		delete(shard.swarms, infoHash)
	}

	return nil
}

func (s *peerStore) PutLeecher(infoHash chihaya.InfoHash, p chihaya.Peer) error {
	select {
	case <-s.closed:
		panic("attempted to interact with stopped store")
	default:
	}

	shard := s.shards[s.shardIndex(infoHash)]
	shard.Lock()
	defer shard.Unlock()

	if _, ok := shard.swarms[infoHash]; !ok {
		shard.swarms[infoHash] = swarm{
			seeders:  make(map[serializedPeer]int64),
			leechers: make(map[serializedPeer]int64),
		}
	}

	shard.swarms[infoHash].leechers[peerKey(p)] = time.Now().UnixNano()

	return nil
}

func (s *peerStore) DeleteLeecher(infoHash chihaya.InfoHash, p chihaya.Peer) error {
	select {
	case <-s.closed:
		panic("attempted to interact with stopped store")
	default:
	}

	shard := s.shards[s.shardIndex(infoHash)]
	pk := peerKey(p)
	shard.Lock()
	defer shard.Unlock()

	if _, ok := shard.swarms[infoHash]; !ok {
		return store.ErrResourceDoesNotExist
	}

	if _, ok := shard.swarms[infoHash].leechers[pk]; !ok {
		return store.ErrResourceDoesNotExist
	}

	delete(shard.swarms[infoHash].leechers, pk)

	if len(shard.swarms[infoHash].seeders)|len(shard.swarms[infoHash].leechers) == 0 {
		delete(shard.swarms, infoHash)
	}

	return nil
}

func (s *peerStore) GraduateLeecher(infoHash chihaya.InfoHash, p chihaya.Peer) error {
	select {
	case <-s.closed:
		panic("attempted to interact with stopped store")
	default:
	}

	key := peerKey(p)
	shard := s.shards[s.shardIndex(infoHash)]
	shard.Lock()
	defer shard.Unlock()

	if _, ok := shard.swarms[infoHash]; !ok {
		shard.swarms[infoHash] = swarm{
			seeders:  make(map[serializedPeer]int64),
			leechers: make(map[serializedPeer]int64),
		}
	}

	delete(shard.swarms[infoHash].leechers, key)

	shard.swarms[infoHash].seeders[key] = time.Now().UnixNano()

	return nil
}

func (s *peerStore) CollectGarbage(cutoff time.Time) error {
	select {
	case <-s.closed:
		panic("attempted to interact with stopped store")
	default:
	}

	log.Printf("memory: collecting garbage. Cutoff time: %s", cutoff.String())
	cutoffUnix := cutoff.UnixNano()
	for _, shard := range s.shards {
		shard.RLock()
		var infohashes []chihaya.InfoHash
		for key := range shard.swarms {
			infohashes = append(infohashes, key)
		}
		shard.RUnlock()
		runtime.Gosched()

		for _, infohash := range infohashes {
			shard.Lock()

			for peerKey, mtime := range shard.swarms[infohash].leechers {
				if mtime <= cutoffUnix {
					delete(shard.swarms[infohash].leechers, peerKey)
				}
			}

			for peerKey, mtime := range shard.swarms[infohash].seeders {
				if mtime <= cutoffUnix {
					delete(shard.swarms[infohash].seeders, peerKey)
				}
			}

			if len(shard.swarms[infohash].seeders)|len(shard.swarms[infohash].leechers) == 0 {
				delete(shard.swarms, infohash)
			}

			shard.Unlock()
			runtime.Gosched()
		}

		runtime.Gosched()
	}

	return nil
}

func (s *peerStore) AnnouncePeers(infoHash chihaya.InfoHash, seeder bool, numWant int, peer4, peer6 chihaya.Peer) (peers, peers6 []chihaya.Peer, err error) {
	select {
	case <-s.closed:
		panic("attempted to interact with stopped store")
	default:
	}

	shard := s.shards[s.shardIndex(infoHash)]
	shard.RLock()
	defer shard.RUnlock()

	if _, ok := shard.swarms[infoHash]; !ok {
		return nil, nil, store.ErrResourceDoesNotExist
	}

	if seeder {
		// Append leechers as possible.
		leechers := shard.swarms[infoHash].leechers
		for p := range leechers {
			decodedPeer := decodePeerKey(p)
			if numWant == 0 {
				break
			}

			if decodedPeer.IP.To4() == nil {
				peers6 = append(peers6, decodedPeer)
			} else {
				peers = append(peers, decodedPeer)
			}
			numWant--
		}
	} else {
		// Append as many seeders as possible.
		seeders := shard.swarms[infoHash].seeders
		for p := range seeders {
			decodedPeer := decodePeerKey(p)
			if numWant == 0 {
				break
			}

			if decodedPeer.IP.To4() == nil {
				peers6 = append(peers6, decodedPeer)
			} else {
				peers = append(peers, decodedPeer)
			}
			numWant--
		}

		// Append leechers until we reach numWant.
		leechers := shard.swarms[infoHash].leechers
		if numWant > 0 {
			for p := range leechers {
				decodedPeer := decodePeerKey(p)
				if numWant == 0 {
					break
				}

				if decodedPeer.IP.To4() == nil {
					if decodedPeer.Equal(peer6) {
						continue
					}
					peers6 = append(peers6, decodedPeer)
				} else {
					if decodedPeer.Equal(peer4) {
						continue
					}
					peers = append(peers, decodedPeer)
				}
				numWant--
			}
		}
	}

	return
}

func (s *peerStore) GetSeeders(infoHash chihaya.InfoHash) (peers, peers6 []chihaya.Peer, err error) {
	select {
	case <-s.closed:
		panic("attempted to interact with stopped store")
	default:
	}

	shard := s.shards[s.shardIndex(infoHash)]
	shard.RLock()
	defer shard.RUnlock()

	if _, ok := shard.swarms[infoHash]; !ok {
		return nil, nil, store.ErrResourceDoesNotExist
	}

	seeders := shard.swarms[infoHash].seeders
	for p := range seeders {
		decodedPeer := decodePeerKey(p)
		if decodedPeer.IP.To4() == nil {
			peers6 = append(peers6, decodedPeer)
		} else {
			peers = append(peers, decodedPeer)
		}
	}
	return
}

func (s *peerStore) GetLeechers(infoHash chihaya.InfoHash) (peers, peers6 []chihaya.Peer, err error) {
	select {
	case <-s.closed:
		panic("attempted to interact with stopped store")
	default:
	}

	shard := s.shards[s.shardIndex(infoHash)]
	shard.RLock()
	defer shard.RUnlock()

	if _, ok := shard.swarms[infoHash]; !ok {
		return nil, nil, store.ErrResourceDoesNotExist
	}

	leechers := shard.swarms[infoHash].leechers
	for p := range leechers {
		decodedPeer := decodePeerKey(p)
		if decodedPeer.IP.To4() == nil {
			peers6 = append(peers6, decodedPeer)
		} else {
			peers = append(peers, decodedPeer)
		}
	}
	return
}

func (s *peerStore) NumSeeders(infoHash chihaya.InfoHash) int {
	select {
	case <-s.closed:
		panic("attempted to interact with stopped store")
	default:
	}

	shard := s.shards[s.shardIndex(infoHash)]
	shard.RLock()
	defer shard.RUnlock()

	if _, ok := shard.swarms[infoHash]; !ok {
		return 0
	}

	return len(shard.swarms[infoHash].seeders)
}

func (s *peerStore) NumLeechers(infoHash chihaya.InfoHash) int {
	select {
	case <-s.closed:
		panic("attempted to interact with stopped store")
	default:
	}

	shard := s.shards[s.shardIndex(infoHash)]
	shard.RLock()
	defer shard.RUnlock()

	if _, ok := shard.swarms[infoHash]; !ok {
		return 0
	}

	return len(shard.swarms[infoHash].leechers)
}

func (s *peerStore) Stop() <-chan error {
	toReturn := make(chan error)
	go func() {
		shards := make([]*peerShard, len(s.shards))
		for i := 0; i < len(s.shards); i++ {
			shards[i] = &peerShard{}
			shards[i].swarms = make(map[chihaya.InfoHash]swarm)
		}
		s.shards = shards
		close(s.closed)
		close(toReturn)
	}()
	return toReturn
}
