package memory

import (
	"encoding/binary"
	"errors"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/chihaya/chihaya/bittorrent"
	"github.com/chihaya/chihaya/storage"
)

func init() {
	prometheus.MustRegister(promGCDurationMilliseconds)
	prometheus.MustRegister(promInfohashesCount)
	prometheus.MustRegister(promSeedersCount, promLeechersCount)
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

var promSeedersCount = prometheus.NewGauge(prometheus.GaugeOpts{
	Name: "chihaya_storage_seeders_count",
	Help: "The number of seeders tracked",
})

var promLeechersCount = prometheus.NewGauge(prometheus.GaugeOpts{
	Name: "chihaya_storage_leechers_count",
	Help: "The number of leechers tracked",
})

// recordGCDuration records the duration of a GC sweep.
func recordGCDuration(duration time.Duration) {
	promGCDurationMilliseconds.Observe(float64(duration.Nanoseconds()) / float64(time.Millisecond))
}

// ErrInvalidGCInterval is returned for a GarbageCollectionInterval that is
// less than or equal to zero.
var ErrInvalidGCInterval = errors.New("invalid garbage collection interval")

// Config holds the configuration of a memory PeerStore.
type Config struct {
	GarbageCollectionInterval   time.Duration `yaml:"gc_interval"`
	PrometheusReportingInterval time.Duration `yaml:"prometheus_reporting_interval"`
	PeerLifetime                time.Duration `yaml:"peer_lifetime"`
	ShardCount                  int           `yaml:"shard_count"`
}

// LogFields renders the current config as a set of Logrus fields.
func (cfg Config) LogFields() log.Fields {
	return log.Fields{
		"gcInterval":   cfg.GarbageCollectionInterval,
		"peerLifetime": cfg.PeerLifetime,
		"shardCount":   cfg.ShardCount,
	}
}

// New creates a new PeerStore backed by memory.
func New(cfg Config) (storage.PeerStore, error) {
	var shardCount int
	if cfg.ShardCount > 0 {
		shardCount = cfg.ShardCount
	} else {
		log.Warnln("storage: shardCount not configured, using 1 as default value.")
		shardCount = 1
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

	ps.wg.Add(1)
	go func() {
		defer ps.wg.Done()
		t := time.NewTicker(1 * time.Second)
		for {
			select {
			case <-ps.closing:
				t.Stop()
				return
			case now := <-t.C:
				ps.setClock(now.UnixNano())
			}
		}
	}()

	ps.wg.Add(1)
	go func() {
		defer ps.wg.Done()
		if cfg.PrometheusReportingInterval <= 0 {
			cfg.PrometheusReportingInterval = 1
			log.Warn("storage: PrometheusReportingInterval not specified/invalid, defaulting to 1 second")
		}
		t := time.NewTicker(cfg.PrometheusReportingInterval)
		for {
			select {
			case <-ps.closing:
				t.Stop()
				return
			case <-t.C:
				before := time.Now()
				ps.populateProm()
				log.Debugf("memory: populateProm() took %s", time.Since(before))
			}
		}
	}()

	return ps, nil
}

type serializedPeer string

type peerShard struct {
	swarms      map[bittorrent.InfoHash]swarm
	numSeeders  uint64
	numLeechers uint64
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
	// clock stores the current time nanoseconds, updated every second.
	// Must be accessed atomically!
	clock int64
	wg    sync.WaitGroup
}

// populateProm aggregates metrics over all shards and then posts them to
// prometheus.
func (ps *peerStore) populateProm() {
	var numInfohashes, numSeeders, numLeechers uint64

	for _, s := range ps.shards {
		s.RLock()
		numInfohashes += uint64(len(s.swarms))
		numSeeders += s.numSeeders
		numLeechers += s.numLeechers
		s.RUnlock()
	}

	promInfohashesCount.Set(float64(numInfohashes))
	promSeedersCount.Set(float64(numSeeders))
	promLeechersCount.Set(float64(numLeechers))
}

var _ storage.PeerStore = &peerStore{}

func (ps *peerStore) getClock() int64 {
	return atomic.LoadInt64(&ps.clock)
}

func (ps *peerStore) setClock(to int64) {
	atomic.StoreInt64(&ps.clock, to)
}

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
	}

	if _, ok := shard.swarms[ih].seeders[pk]; !ok {
		// new peer
		shard.numSeeders++
	}
	shard.swarms[ih].seeders[pk] = ps.getClock()

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

	if _, ok := shard.swarms[ih].seeders[pk]; ok {
		// seeder actually removed
		shard.numSeeders--
		delete(shard.swarms[ih].seeders, pk)
	}

	if len(shard.swarms[ih].seeders)|len(shard.swarms[ih].leechers) == 0 {
		delete(shard.swarms, ih)
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
	}

	if _, ok := shard.swarms[ih].leechers[pk]; !ok {
		// new leecher
		shard.numLeechers++
	}
	shard.swarms[ih].leechers[pk] = ps.getClock()

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

	if _, ok := shard.swarms[ih].leechers[pk]; ok {
		// leecher actually removed
		shard.numLeechers--
		delete(shard.swarms[ih].leechers, pk)
	}

	if len(shard.swarms[ih].seeders)|len(shard.swarms[ih].leechers) == 0 {
		delete(shard.swarms, ih)
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
	}

	if _, ok := shard.swarms[ih].leechers[pk]; ok {
		// leecher actually removed
		shard.numLeechers--
		delete(shard.swarms[ih].leechers, pk)
	}

	if _, ok := shard.swarms[ih].seeders[pk]; !ok {
		// new seeder
		shard.numSeeders++
	}
	shard.swarms[ih].seeders[pk] = ps.getClock()

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
					shard.numLeechers--
				}
			}

			for pk, mtime := range shard.swarms[ih].seeders {
				if mtime <= cutoffUnix {
					delete(shard.swarms[ih].seeders, pk)
					shard.numSeeders--
				}
			}

			if len(shard.swarms[ih].seeders)|len(shard.swarms[ih].leechers) == 0 {
				delete(shard.swarms, ih)
			}

			shard.Unlock()
			runtime.Gosched()
		}

		runtime.Gosched()
	}

	recordGCDuration(time.Since(start))

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
