package memory

import (
	"encoding/binary"
	"log"
	"net"
	"runtime"
	"sync"
	"time"

	"github.com/jzelinskie/trakr/bittorrent"
	"github.com/jzelinskie/trakr/storage"
)

// TODO(jzelinskie): separate ipv4 and ipv6 swarms

type Config struct {
	ShardCount int `yaml:"shard_count"`
}

func New(cfg Config) (storage.PeerStore, error) {
	shardCount := 1
	if cfg.ShardCount > 0 {
		shardCount = cfg.ShardCount
	}

	shards := make([]*peerShard, shardCount)
	for i := 0; i < shardCount; i++ {
		shards[i] = &peerShard{}
		shards[i].swarms = make(map[swarmKey]swarm)
	}

	return &peerStore{
		shards: shards,
		closed: make(chan struct{}),
	}, nil
}

type serializedPeer string

type swarmKey [21]byte

func newSwarmKey(ih bittorrent.InfoHash, p bittorrent.Peer) (key swarmKey) {
	for i, ihbyte := range ih {
		key[i] = ihbyte
	}
	if len(p.IP) == net.IPv4len {
		key[20] = byte(4)
	} else {
		key[20] = byte(6)
	}

	return
}

type peerShard struct {
	swarms map[swarmKey]swarm
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

var _ storage.PeerStore = &peerStore{}

func (s *peerStore) shardIndex(infoHash bittorrent.InfoHash) uint32 {
	return binary.BigEndian.Uint32(infoHash[:4]) % uint32(len(s.shards))
}

func newPeerKey(p bittorrent.Peer) serializedPeer {
	b := make([]byte, 20+2+len(p.IP))
	copy(b[:20], p.ID[:])
	binary.BigEndian.PutUint16(b[20:22], p.Port)
	copy(b[22:], p.IP)

	return serializedPeer(b)
}

func decodePeerKey(pk serializedPeer) bittorrent.Peer {
	return bittorrent.Peer{
		ID:   bittorrent.PeerIDFromString(string(pk[:20])),
		Port: binary.BigEndian.Uint16([]byte(pk[20:22])),
		IP:   net.IP(pk[22:]),
	}
}

func (s *peerStore) PutSeeder(ih bittorrent.InfoHash, p bittorrent.Peer) error {
	select {
	case <-s.closed:
		panic("attempted to interact with stopped memory store")
	default:
	}

	sk := newSwarmKey(ih, p)
	pk := newPeerKey(p)

	shard := s.shards[s.shardIndex(ih)]
	shard.Lock()

	if _, ok := shard.swarms[sk]; !ok {
		shard.swarms[sk] = swarm{
			seeders:  make(map[serializedPeer]int64),
			leechers: make(map[serializedPeer]int64),
		}
	}

	shard.swarms[sk].seeders[pk] = time.Now().UnixNano()

	shard.Unlock()
	return nil
}

func (s *peerStore) DeleteSeeder(ih bittorrent.InfoHash, p bittorrent.Peer) error {
	select {
	case <-s.closed:
		panic("attempted to interact with stopped memory store")
	default:
	}

	sk := newSwarmKey(ih, p)
	pk := newPeerKey(p)

	shard := s.shards[s.shardIndex(ih)]
	shard.Lock()

	if _, ok := shard.swarms[sk]; !ok {
		shard.Unlock()
		return storage.ErrResourceDoesNotExist
	}

	if _, ok := shard.swarms[sk].seeders[pk]; !ok {
		shard.Unlock()
		return storage.ErrResourceDoesNotExist
	}

	delete(shard.swarms[sk].seeders, pk)

	if len(shard.swarms[sk].seeders)|len(shard.swarms[sk].leechers) == 0 {
		delete(shard.swarms, sk)
	}

	shard.Unlock()
	return nil
}

func (s *peerStore) PutLeecher(ih bittorrent.InfoHash, p bittorrent.Peer) error {
	select {
	case <-s.closed:
		panic("attempted to interact with stopped memory store")
	default:
	}

	sk := newSwarmKey(ih, p)
	pk := newPeerKey(p)

	shard := s.shards[s.shardIndex(ih)]
	shard.Lock()

	if _, ok := shard.swarms[sk]; !ok {
		shard.swarms[sk] = swarm{
			seeders:  make(map[serializedPeer]int64),
			leechers: make(map[serializedPeer]int64),
		}
	}

	shard.swarms[sk].leechers[pk] = time.Now().UnixNano()

	shard.Unlock()
	return nil
}

func (s *peerStore) DeleteLeecher(ih bittorrent.InfoHash, p bittorrent.Peer) error {
	select {
	case <-s.closed:
		panic("attempted to interact with stopped memory store")
	default:
	}

	sk := newSwarmKey(ih, p)
	pk := newPeerKey(p)

	shard := s.shards[s.shardIndex(ih)]
	shard.Lock()

	if _, ok := shard.swarms[sk]; !ok {
		shard.Unlock()
		return storage.ErrResourceDoesNotExist
	}

	if _, ok := shard.swarms[sk].leechers[pk]; !ok {
		shard.Unlock()
		return storage.ErrResourceDoesNotExist
	}

	delete(shard.swarms[sk].leechers, pk)

	if len(shard.swarms[sk].seeders)|len(shard.swarms[sk].leechers) == 0 {
		delete(shard.swarms, sk)
	}

	shard.Unlock()
	return nil
}

func (s *peerStore) GraduateLeecher(ih bittorrent.InfoHash, p bittorrent.Peer) error {
	select {
	case <-s.closed:
		panic("attempted to interact with stopped memory store")
	default:
	}

	sk := newSwarmKey(ih, p)
	pk := newPeerKey(p)

	shard := s.shards[s.shardIndex(ih)]
	shard.Lock()

	if _, ok := shard.swarms[sk]; !ok {
		shard.swarms[sk] = swarm{
			seeders:  make(map[serializedPeer]int64),
			leechers: make(map[serializedPeer]int64),
		}
	}

	delete(shard.swarms[sk].leechers, pk)

	shard.swarms[sk].seeders[pk] = time.Now().UnixNano()

	shard.Unlock()
	return nil
}

func (s *peerStore) CollectGarbage(cutoff time.Time) error {
	select {
	case <-s.closed:
		panic("attempted to interact with stopped memory store")
	default:
	}

	log.Printf("memory: collecting garbage. Cutoff time: %s", cutoff.String())
	cutoffUnix := cutoff.UnixNano()
	for _, shard := range s.shards {
		shard.RLock()
		var swarmKeys []swarmKey
		for sk := range shard.swarms {
			swarmKeys = append(swarmKeys, sk)
		}
		shard.RUnlock()
		runtime.Gosched()

		for _, sk := range swarmKeys {
			shard.Lock()

			if _, stillExists := shard.swarms[sk]; !stillExists {
				shard.Unlock()
				runtime.Gosched()
				continue
			}

			for pk, mtime := range shard.swarms[sk].leechers {
				if mtime <= cutoffUnix {
					delete(shard.swarms[sk].leechers, pk)
				}
			}

			for pk, mtime := range shard.swarms[sk].seeders {
				if mtime <= cutoffUnix {
					delete(shard.swarms[sk].seeders, pk)
				}
			}

			if len(shard.swarms[sk].seeders)|len(shard.swarms[sk].leechers) == 0 {
				delete(shard.swarms, sk)
			}

			shard.Unlock()
			runtime.Gosched()
		}

		runtime.Gosched()
	}

	return nil
}

func (s *peerStore) AnnouncePeers(ih bittorrent.InfoHash, seeder bool, numWant int, announcer bittorrent.Peer) (peers []bittorrent.Peer, err error) {
	select {
	case <-s.closed:
		panic("attempted to interact with stopped memory store")
	default:
	}

	sk := newSwarmKey(ih, announcer)

	shard := s.shards[s.shardIndex(ih)]
	shard.RLock()

	if _, ok := shard.swarms[sk]; !ok {
		shard.RUnlock()
		return nil, storage.ErrResourceDoesNotExist
	}

	if seeder {
		// Append leechers as possible.
		leechers := shard.swarms[sk].leechers
		for p := range leechers {
			decodedPeer := decodePeerKey(p)
			if numWant == 0 {
				break
			}

			peers = append(peers, decodedPeer)
			numWant--
		}
	} else {
		// Append as many seeders as possible.
		seeders := shard.swarms[sk].seeders
		for p := range seeders {
			decodedPeer := decodePeerKey(p)
			if numWant == 0 {
				break
			}

			peers = append(peers, decodedPeer)
			numWant--
		}

		// Append leechers until we reach numWant.
		leechers := shard.swarms[sk].leechers
		if numWant > 0 {
			for p := range leechers {
				decodedPeer := decodePeerKey(p)
				if numWant == 0 {
					break
				}

				if decodedPeer.Equal(announcer) {
					continue
				}
				peers = append(peers, decodedPeer)
				numWant--
			}
		}
	}

	shard.RUnlock()
	return
}

func (s *peerStore) Stop() <-chan error {
	toReturn := make(chan error)
	go func() {
		shards := make([]*peerShard, len(s.shards))
		for i := 0; i < len(s.shards); i++ {
			shards[i] = &peerShard{}
			shards[i].swarms = make(map[swarmKey]swarm)
		}
		s.shards = shards
		close(s.closed)
		close(toReturn)
	}()
	return toReturn
}
