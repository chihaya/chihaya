// Package memorybysubnet implements the storage interface for a Chihaya
// BitTorrent tracker keeping peer data in memory organized by a pre-configured
// subnet.
package memorybysubnet

import (
	"encoding/binary"
	"errors"
	"net"
	"runtime"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/yaml.v2"

	"github.com/chihaya/chihaya/bittorrent"
	"github.com/chihaya/chihaya/storage"
)

func init() {
	prometheus.MustRegister(promGCDurationMilliseconds)
	prometheus.MustRegister(promInfohashesCount)

	// Register the storage driver.
	storage.RegisterDriver("memorybysubnet", driver{})
}

var promGCDurationMilliseconds = prometheus.NewHistogram(prometheus.HistogramOpts{
	Name:    "chihaya_storage_gc_duration_milliseconds",
	Help:    "The time it takes to perform storage garbage collection",
	Buckets: prometheus.ExponentialBuckets(9.375, 2, 10),
})

var promInfohashesCount = prometheus.NewGauge(prometheus.GaugeOpts{
	Name: "chihaya_storage_infohashes_count",
	Help: "The number of Infohashes tracked",
})

// recordGCDuration records the duration of a GC sweep.
func recordGCDuration(duration time.Duration) {
	promGCDurationMilliseconds.Observe(float64(duration.Nanoseconds()) / float64(time.Millisecond))
}

// recordInfohashesDelta records a change in the number of Infohashes tracked.
func recordInfohashesDelta(delta float64) {
	promInfohashesCount.Add(delta)
}

type driver struct{}

func (d driver) NewPeerStore(icfg interface{}) (storage.PeerStore, error) {
	// Marshal the config back into bytes.
	bytes, err := yaml.Marshal(icfg)
	if err != nil {
		return nil, err
	}

	// Unmarshal the bytes into the proper config type.
	var cfg Config
	err = yaml.Unmarshal(bytes, &cfg)
	if err != nil {
		return nil, err
	}

	return New(cfg)
}

// ErrInvalidGCInterval is returned for a GarbageCollectionInterval that is
// less than or equal to zero.
var ErrInvalidGCInterval = errors.New("invalid garbage collection interval")

// Config holds the configuration of a memory PeerStore.
type Config struct {
	GarbageCollectionInterval      time.Duration `yaml:"gc_interval"`
	PeerLifetime                   time.Duration `yaml:"peer_lifetime"`
	ShardCount                     int           `yaml:"shard_count"`
	PreferredIPv4SubnetMaskBitsSet int           `yaml:"preferred_ipv4_subnet_mask_bits_set"`
	PreferredIPv6SubnetMaskBitsSet int           `yaml:"preferred_ipv6_subnet_mask_bits_set"`
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
		shards:   make([]*peerShard, shardCount*2),
		closed:   make(chan struct{}),
		ipv4Mask: net.CIDRMask(cfg.PreferredIPv4SubnetMaskBitsSet, 32),
		ipv6Mask: net.CIDRMask(cfg.PreferredIPv6SubnetMaskBitsSet, 128),
	}

	for i := 0; i < shardCount*2; i++ {
		ps.shards[i] = &peerShard{swarms: make(map[bittorrent.InfoHash]swarm)}
	}

	go func() {
		for {
			select {
			case <-ps.closed:
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
	seeders  map[string]map[serializedPeer]int64
	leechers map[string]map[serializedPeer]int64
}

func (s swarm) lenSeeders() (i int) {
	for _, subnet := range s.seeders {
		i += len(subnet)
	}
	return
}

func (s swarm) lenLeechers() (i int) {
	for _, subnet := range s.leechers {
		i += len(subnet)
	}
	return
}

type peerStore struct {
	shards   []*peerShard
	closed   chan struct{}
	ipv4Mask net.IPMask
	ipv6Mask net.IPMask
}

var _ storage.PeerStore = &peerStore{}

func (s *peerStore) shardIndex(infoHash bittorrent.InfoHash, af bittorrent.AddressFamily) uint32 {
	// There are twice the amount of shards specified by the user, the first
	// half is dedicated to IPv4 swarms and the second half is dedicated to
	// IPv6 swarms.
	idx := binary.BigEndian.Uint32(infoHash[:4]) % (uint32(len(s.shards)) / 2)
	if af == bittorrent.IPv6 {
		idx += uint32(len(s.shards) / 2)
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

func (s *peerStore) mask(p bittorrent.Peer) string {
	var maskedIP net.IP
	switch p.IP.AddressFamily {
	case bittorrent.IPv4:
		maskedIP = p.IP.IP.Mask(s.ipv4Mask)
	case bittorrent.IPv6:
		maskedIP = p.IP.IP.Mask(s.ipv6Mask)
	default:
		panic("IP is neither v4 nor v6")
	}
	return maskedIP.String()
}

func (s *peerStore) PutSeeder(ih bittorrent.InfoHash, p bittorrent.Peer) error {
	select {
	case <-s.closed:
		panic("attempted to interact with stopped memory store")
	default:
	}

	pk := newPeerKey(p)

	shard := s.shards[s.shardIndex(ih, p.IP.AddressFamily)]
	shard.Lock()

	if _, ok := shard.swarms[ih]; !ok {
		shard.swarms[ih] = swarm{
			seeders:  make(map[string]map[serializedPeer]int64),
			leechers: make(map[string]map[serializedPeer]int64),
		}
		recordInfohashesDelta(1)
	}

	preferredSubnet := s.mask(p)
	if shard.swarms[ih].seeders[preferredSubnet] == nil {
		shard.swarms[ih].seeders[preferredSubnet] = make(map[serializedPeer]int64)
	}
	shard.swarms[ih].seeders[preferredSubnet][pk] = time.Now().UnixNano()

	shard.Unlock()
	return nil
}

func (s *peerStore) DeleteSeeder(ih bittorrent.InfoHash, p bittorrent.Peer) error {
	select {
	case <-s.closed:
		panic("attempted to interact with stopped memory store")
	default:
	}

	pk := newPeerKey(p)

	shard := s.shards[s.shardIndex(ih, p.IP.AddressFamily)]
	shard.Lock()

	if _, ok := shard.swarms[ih]; !ok {
		shard.Unlock()
		return storage.ErrResourceDoesNotExist
	}

	preferredSubnet := s.mask(p)
	if _, ok := shard.swarms[ih].seeders[preferredSubnet][pk]; !ok {
		shard.Unlock()
		return storage.ErrResourceDoesNotExist
	}

	delete(shard.swarms[ih].seeders[preferredSubnet], pk)

	if shard.swarms[ih].lenSeeders()|shard.swarms[ih].lenLeechers() == 0 {
		delete(shard.swarms, ih)
		recordInfohashesDelta(-1)
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

	pk := newPeerKey(p)

	shard := s.shards[s.shardIndex(ih, p.IP.AddressFamily)]
	shard.Lock()

	if _, ok := shard.swarms[ih]; !ok {
		shard.swarms[ih] = swarm{
			seeders:  make(map[string]map[serializedPeer]int64),
			leechers: make(map[string]map[serializedPeer]int64),
		}
		recordInfohashesDelta(1)
	}

	preferredSubnet := s.mask(p)
	if shard.swarms[ih].leechers[preferredSubnet] == nil {
		shard.swarms[ih].leechers[preferredSubnet] = make(map[serializedPeer]int64)
	}
	shard.swarms[ih].leechers[preferredSubnet][pk] = time.Now().UnixNano()

	shard.Unlock()
	return nil
}

func (s *peerStore) DeleteLeecher(ih bittorrent.InfoHash, p bittorrent.Peer) error {
	select {
	case <-s.closed:
		panic("attempted to interact with stopped memory store")
	default:
	}

	pk := newPeerKey(p)

	shard := s.shards[s.shardIndex(ih, p.IP.AddressFamily)]
	shard.Lock()

	if _, ok := shard.swarms[ih]; !ok {
		shard.Unlock()
		return storage.ErrResourceDoesNotExist
	}

	preferredSubnet := s.mask(p)
	if _, ok := shard.swarms[ih].leechers[preferredSubnet][pk]; !ok {
		shard.Unlock()
		return storage.ErrResourceDoesNotExist
	}

	delete(shard.swarms[ih].leechers[preferredSubnet], pk)

	if shard.swarms[ih].lenSeeders()|shard.swarms[ih].lenLeechers() == 0 {
		delete(shard.swarms, ih)
		recordInfohashesDelta(-1)
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

	pk := newPeerKey(p)

	shard := s.shards[s.shardIndex(ih, p.IP.AddressFamily)]
	shard.Lock()

	if _, ok := shard.swarms[ih]; !ok {
		shard.swarms[ih] = swarm{
			seeders:  make(map[string]map[serializedPeer]int64),
			leechers: make(map[string]map[serializedPeer]int64),
		}
		recordInfohashesDelta(1)
	}

	preferredSubnet := s.mask(p)
	delete(shard.swarms[ih].leechers[preferredSubnet], pk)

	if shard.swarms[ih].seeders[preferredSubnet] == nil {
		shard.swarms[ih].seeders[preferredSubnet] = make(map[serializedPeer]int64)
	}
	shard.swarms[ih].seeders[preferredSubnet][pk] = time.Now().UnixNano()

	shard.Unlock()
	return nil
}

func (s *peerStore) AnnouncePeers(ih bittorrent.InfoHash, seeder bool, numWant int, announcer bittorrent.Peer) (peers []bittorrent.Peer, err error) {
	select {
	case <-s.closed:
		panic("attempted to interact with stopped memory store")
	default:
	}

	shard := s.shards[s.shardIndex(ih, announcer.IP.AddressFamily)]
	shard.RLock()

	if _, ok := shard.swarms[ih]; !ok {
		shard.RUnlock()
		return nil, storage.ErrResourceDoesNotExist
	}

	preferredSubnet := s.mask(announcer)

	if seeder {
		// Append as many close leechers as possible.
		closestLeechers := shard.swarms[ih].leechers[preferredSubnet]
		for p := range closestLeechers {
			if numWant == 0 {
				break
			}
			decodedPeer := decodePeerKey(p)

			peers = append(peers, decodedPeer)
			numWant--
		}

		// Append the rest of the leechers.
		if numWant > 0 {
			for subnet := range shard.swarms[ih].leechers {
				if subnet == preferredSubnet {
					continue
				}

				for p := range shard.swarms[ih].leechers[subnet] {
					if numWant == 0 {
						break
					}
					decodedPeer := decodePeerKey(p)

					peers = append(peers, decodedPeer)
					numWant--
				}
			}
		}
	} else {
		// Append as many close seeders as possible.
		closestSeeders := shard.swarms[ih].seeders[preferredSubnet]
		for p := range closestSeeders {
			if numWant == 0 {
				break
			}
			decodedPeer := decodePeerKey(p)

			peers = append(peers, decodedPeer)
			numWant--
		}

		// Append as many close leechers as possible.
		closestLeechers := shard.swarms[ih].leechers[preferredSubnet]
		for p := range closestLeechers {
			if numWant == 0 {
				break
			}
			decodedPeer := decodePeerKey(p)

			peers = append(peers, decodedPeer)
			numWant--
		}

		// Append as the rest of the seeders.
		if numWant > 0 {
			for subnet := range shard.swarms[ih].seeders {
				if subnet == preferredSubnet {
					continue
				}

				for p := range shard.swarms[ih].seeders[subnet] {
					if numWant == 0 {
						break
					}
					decodedPeer := decodePeerKey(p)

					peers = append(peers, decodedPeer)
					numWant--
				}
			}
		}

		// Append the rest of the leechers.
		if numWant > 0 {
			for subnet := range shard.swarms[ih].leechers {
				if subnet == preferredSubnet {
					continue
				}

				for p := range shard.swarms[ih].leechers[subnet] {
					if numWant == 0 {
						break
					}
					decodedPeer := decodePeerKey(p)

					if decodedPeer.Equal(announcer) {
						continue
					}
					peers = append(peers, decodedPeer)
					numWant--
				}
			}
		}
	}

	shard.RUnlock()
	return
}

func (s *peerStore) ScrapeSwarm(ih bittorrent.InfoHash, addressFamily bittorrent.AddressFamily) (resp bittorrent.Scrape) {
	select {
	case <-s.closed:
		panic("attempted to interact with stopped memory store")
	default:
	}

	resp.InfoHash = ih
	shard := s.shards[s.shardIndex(ih, addressFamily)]
	shard.RLock()

	if _, ok := shard.swarms[ih]; !ok {
		shard.RUnlock()
		return
	}

	resp.Incomplete = uint32(shard.swarms[ih].lenLeechers())
	resp.Complete = uint32(shard.swarms[ih].lenSeeders())
	shard.RUnlock()

	return
}

// collectGarbage deletes all Peers from the PeerStore which are older than the
// cutoff time.
//
// This function must be able to execute while other methods on this interface
// are being executed in parallel.
func (s *peerStore) collectGarbage(cutoff time.Time) error {
	select {
	case <-s.closed:
		panic("attempted to interact with stopped memory store")
	default:
	}

	var ihDelta float64
	cutoffUnix := cutoff.UnixNano()
	start := time.Now()

	for _, shard := range s.shards {
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

			for subnet := range shard.swarms[ih].leechers {
				for pk, mtime := range shard.swarms[ih].leechers[subnet] {
					if mtime <= cutoffUnix {
						delete(shard.swarms[ih].leechers[subnet], pk)
					}
				}
				if len(shard.swarms[ih].leechers[subnet]) == 0 {
					delete(shard.swarms[ih].leechers, subnet)
				}
			}

			for subnet := range shard.swarms[ih].seeders {
				for pk, mtime := range shard.swarms[ih].seeders[subnet] {
					if mtime <= cutoffUnix {
						delete(shard.swarms[ih].seeders[subnet], pk)
					}
				}
				if len(shard.swarms[ih].seeders[subnet]) == 0 {
					delete(shard.swarms[ih].seeders, subnet)
				}
			}

			// TODO(jzelinskie): fix this to sum all peers in all subnets
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

func (s *peerStore) Stop() <-chan error {
	toReturn := make(chan error)
	go func() {
		shards := make([]*peerShard, len(s.shards))
		for i := 0; i < len(s.shards); i++ {
			shards[i] = &peerShard{swarms: make(map[bittorrent.InfoHash]swarm)}
		}
		s.shards = shards
		close(s.closed)
		close(toReturn)
	}()
	return toReturn
}
