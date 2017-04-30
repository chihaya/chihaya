package memory

import (
	"encoding/binary"
	"errors"
	"net"
	"runtime"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/chihaya/chihaya/bittorrent"
	"github.com/chihaya/chihaya/storage"
)

func init() {
	prometheus.MustRegister(promGCDurationMilliseconds)
	prometheus.MustRegister(promInfohashesCount)
}

var promGCDurationMilliseconds = prometheus.NewHistogram(prometheus.HistogramOpts{
	Name:    "chihaya_storage_gc_duration_milliseconds",
	Help:    "The time it takes to perform storage garbage collection",
	Buckets: prometheus.ExponentialBuckets(9.375, 2, 10),
})

var promInfohashesCount = prometheus.NewGauge(prometheus.GaugeOpts{
	Name: "chihaya_storage_infohashes_count",
	Help: "The number of infohashes tracked",
})

// recordGCDuration records the duration of a GC sweep.
func recordGCDuration(duration time.Duration) {
	promGCDurationMilliseconds.Observe(float64(duration.Nanoseconds()) / float64(time.Millisecond))
}

// recordInfohashesDelta records a change in the number of Infohashes tracked.
func recordInfohashesDelta(delta float64) {
	promInfohashesCount.Add(delta)
}

// ErrInvalidGCInterval is returned for a GarbageCollectionInterval that is
// less than or equal to zero.
var ErrInvalidGCInterval = errors.New("invalid garbage collection interval")

// Config holds the configuration of a memory PeerStore.
type Config struct {
	GarbageCollectionInterval time.Duration `yaml:"gc_interval"`
	PeerLifetime              time.Duration `yaml:"peer_lifetime"`
	ShardCount                int           `yaml:"shard_count"`
}

// New creates a new PeerStore backed by memory.
func New(cfg Config) (storage.PeerStore, error) {
	shardCount := 1
	if cfg.ShardCount > 0 {
		shardCount = cfg.ShardCount
	}

	if cfg.GarbageCollectionInterval <= 0 {
		return nil, ErrInvalidGCInterval
	}

	ps := &peerStore{
		shards:  make([]*peerShard, shardCount*2),
		closing: make(chan struct{}),
	}

	for i := 0; i < shardCount*2; i++ {
		ps.shards[i] = &peerShard{swarms: make(map[bittorrent.InfoHash]swarm)}
	}

	ps.wg.Add(1)
	go func() {
		defer ps.wg.Done()
		for {
			select {
			case <-ps.closing:
				return
			case <-time.After(cfg.GarbageCollectionInterval):
				before := time.Now().Add(-cfg.PeerLifetime)
				log.Debugln("memory: purging peers with no announces since", before)
				ps.collectGarbage(before)
			}
		}
	}()

	return ps, nil
}

type serializedPeer string

type peerShard struct {
	swarms map[bittorrent.InfoHash]swarm
	sync.RWMutex
}

type swarm struct {
	// map serialized peer to mtime
	seeders  map[serializedPeer]int64
	leechers map[serializedPeer]int64
}

type peerStore struct {
	shards  []*peerShard
	closing chan struct{}
	wg      sync.WaitGroup
}

var _ storage.PeerStore = &peerStore{}

func (ps *peerStore) shardIndex(infoHash bittorrent.InfoHash, af bittorrent.AddressFamily) uint32 {
	// There are twice the amount of shards specified by the user, the first
	// half is dedicated to IPv4 swarms and the second half is dedicated to
	// IPv6 swarms.
	idx := binary.BigEndian.Uint32(infoHash[:4]) % (uint32(len(ps.shards)) / 2)
	if af == bittorrent.IPv6 {
		idx += uint32(len(ps.shards) / 2)
	}
	return idx
}

func newPeerKey(p bittorrent.Peer) serializedPeer {
	b := make([]byte, 20+2+len(p.IP.IP))
	copy(b[:20], p.ID[:])
	binary.BigEndian.PutUint16(b[20:22], p.Port)
	copy(b[22:], p.IP.IP)

	return serializedPeer(b)
}

func decodePeerKey(pk serializedPeer) bittorrent.Peer {
	peer := bittorrent.Peer{
		ID:   bittorrent.PeerIDFromString(string(pk[:20])),
		Port: binary.BigEndian.Uint16([]byte(pk[20:22])),
		IP:   bittorrent.IP{IP: net.IP(pk[22:])}}

	if ip := peer.IP.To4(); ip != nil {
		peer.IP.IP = ip
		peer.IP.AddressFamily = bittorrent.IPv4
	} else if len(peer.IP.IP) == net.IPv6len { // implies toReturn.IP.To4() == nil
		peer.IP.AddressFamily = bittorrent.IPv6
	} else {
		panic("IP is neither v4 nor v6")
	}

	return peer
}

func (ps *peerStore) PutSeeder(ih bittorrent.InfoHash, p bittorrent.Peer) error {
	select {
	case <-ps.closing:
		panic("attempted to interact with stopped memory store")
	default:
	}

	pk := newPeerKey(p)

	shard := ps.shards[ps.shardIndex(ih, p.IP.AddressFamily)]
	shard.Lock()

	if _, ok := shard.swarms[ih]; !ok {
		shard.swarms[ih] = swarm{
			seeders:  make(map[serializedPeer]int64),
			leechers: make(map[serializedPeer]int64),
		}
		recordInfohashesDelta(1)
	}

	shard.swarms[ih].seeders[pk] = time.Now().UnixNano()

	shard.Unlock()
	return nil
}

func (ps *peerStore) DeleteSeeder(ih bittorrent.InfoHash, p bittorrent.Peer) error {
	select {
	case <-ps.closing:
		panic("attempted to interact with stopped memory store")
	default:
	}

	pk := newPeerKey(p)

	shard := ps.shards[ps.shardIndex(ih, p.IP.AddressFamily)]
	shard.Lock()

	if _, ok := shard.swarms[ih]; !ok {
		shard.Unlock()
		return storage.ErrResourceDoesNotExist
	}

	if _, ok := shard.swarms[ih].seeders[pk]; !ok {
		shard.Unlock()
		return storage.ErrResourceDoesNotExist
	}

	delete(shard.swarms[ih].seeders, pk)

	if len(shard.swarms[ih].seeders)|len(shard.swarms[ih].leechers) == 0 {
		delete(shard.swarms, ih)
		recordInfohashesDelta(-1)
	}

	shard.Unlock()
	return nil
}

func (ps *peerStore) PutLeecher(ih bittorrent.InfoHash, p bittorrent.Peer) error {
	select {
	case <-ps.closing:
		panic("attempted to interact with stopped memory store")
	default:
	}

	pk := newPeerKey(p)

	shard := ps.shards[ps.shardIndex(ih, p.IP.AddressFamily)]
	shard.Lock()

	if _, ok := shard.swarms[ih]; !ok {
		shard.swarms[ih] = swarm{
			seeders:  make(map[serializedPeer]int64),
			leechers: make(map[serializedPeer]int64),
		}
		recordInfohashesDelta(1)
	}

	shard.swarms[ih].leechers[pk] = time.Now().UnixNano()

	shard.Unlock()
	return nil
}

func (ps *peerStore) DeleteLeecher(ih bittorrent.InfoHash, p bittorrent.Peer) error {
	select {
	case <-ps.closing:
		panic("attempted to interact with stopped memory store")
	default:
	}

	pk := newPeerKey(p)

	shard := ps.shards[ps.shardIndex(ih, p.IP.AddressFamily)]
	shard.Lock()

	if _, ok := shard.swarms[ih]; !ok {
		shard.Unlock()
		return storage.ErrResourceDoesNotExist
	}

	if _, ok := shard.swarms[ih].leechers[pk]; !ok {
		shard.Unlock()
		return storage.ErrResourceDoesNotExist
	}

	delete(shard.swarms[ih].leechers, pk)

	if len(shard.swarms[ih].seeders)|len(shard.swarms[ih].leechers) == 0 {
		delete(shard.swarms, ih)
		recordInfohashesDelta(-1)
	}

	shard.Unlock()
	return nil
}

func (ps *peerStore) GraduateLeecher(ih bittorrent.InfoHash, p bittorrent.Peer) error {
	select {
	case <-ps.closing:
		panic("attempted to interact with stopped memory store")
	default:
	}

	pk := newPeerKey(p)

	shard := ps.shards[ps.shardIndex(ih, p.IP.AddressFamily)]
	shard.Lock()

	if _, ok := shard.swarms[ih]; !ok {
		shard.swarms[ih] = swarm{
			seeders:  make(map[serializedPeer]int64),
			leechers: make(map[serializedPeer]int64),
		}
		recordInfohashesDelta(1)
	}

	delete(shard.swarms[ih].leechers, pk)

	shard.swarms[ih].seeders[pk] = time.Now().UnixNano()

	shard.Unlock()
	return nil
}

func (ps *peerStore) AnnouncePeers(ih bittorrent.InfoHash, seeder bool, numWant int, announcer bittorrent.Peer) (peers []bittorrent.Peer, err error) {
	select {
	case <-ps.closing:
		panic("attempted to interact with stopped memory store")
	default:
	}

	shard := ps.shards[ps.shardIndex(ih, announcer.IP.AddressFamily)]
	shard.RLock()

	if _, ok := shard.swarms[ih]; !ok {
		shard.RUnlock()
		return nil, storage.ErrResourceDoesNotExist
	}

	if seeder {
		// Append leechers as possible.
		leechers := shard.swarms[ih].leechers
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
		seeders := shard.swarms[ih].seeders
		for p := range seeders {
			decodedPeer := decodePeerKey(p)
			if numWant == 0 {
				break
			}

			peers = append(peers, decodedPeer)
			numWant--
		}

		// Append leechers until we reach numWant.
		leechers := shard.swarms[ih].leechers
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

func (ps *peerStore) ScrapeSwarm(ih bittorrent.InfoHash, addressFamily bittorrent.AddressFamily) (resp bittorrent.Scrape) {
	select {
	case <-ps.closing:
		panic("attempted to interact with stopped memory store")
	default:
	}

	resp.InfoHash = ih
	shard := ps.shards[ps.shardIndex(ih, addressFamily)]
	shard.RLock()

	if _, ok := shard.swarms[ih]; !ok {
		shard.RUnlock()
		return
	}

	resp.Incomplete = uint32(len(shard.swarms[ih].leechers))
	resp.Complete = uint32(len(shard.swarms[ih].seeders))
	shard.RUnlock()

	return
}

// collectGarbage deletes all Peers from the PeerStore which are older than the
// cutoff time.
//
// This function must be able to execute while other methods on this interface
// are being executed in parallel.
func (ps *peerStore) collectGarbage(cutoff time.Time) error {
	select {
	case <-ps.closing:
		panic("attempted to interact with stopped memory store")
	default:
	}

	var ihDelta float64
	cutoffUnix := cutoff.UnixNano()
	start := time.Now()
	for _, shard := range ps.shards {
		shard.RLock()
		var infohashes []bittorrent.InfoHash
		for ih := range shard.swarms {
			infohashes = append(infohashes, ih)
		}
		shard.RUnlock()
		runtime.Gosched()

		for _, ih := range infohashes {
			shard.Lock()

			if _, stillExists := shard.swarms[ih]; !stillExists {
				shard.Unlock()
				runtime.Gosched()
				continue
			}

			for pk, mtime := range shard.swarms[ih].leechers {
				if mtime <= cutoffUnix {
					delete(shard.swarms[ih].leechers, pk)
				}
			}

			for pk, mtime := range shard.swarms[ih].seeders {
				if mtime <= cutoffUnix {
					delete(shard.swarms[ih].seeders, pk)
				}
			}

			if len(shard.swarms[ih].seeders)|len(shard.swarms[ih].leechers) == 0 {
				delete(shard.swarms, ih)
				ihDelta--
			}

			shard.Unlock()
			runtime.Gosched()
		}

		runtime.Gosched()
	}

	recordGCDuration(time.Since(start))
	recordInfohashesDelta(ihDelta)

	return nil
}

func (ps *peerStore) Stop() <-chan error {
	c := make(chan error)
	go func() {
		close(ps.closing)
		ps.wg.Wait()

		// Explicitly deallocate our storage.
		shards := make([]*peerShard, len(ps.shards))
		for i := 0; i < len(ps.shards); i++ {
			shards[i] = &peerShard{swarms: make(map[bittorrent.InfoHash]swarm)}
		}
		ps.shards = shards

		close(c)
	}()

	return c
}
