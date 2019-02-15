// Package memory implements the storage interface for a Chihaya
// BitTorrent tracker keeping peer data in memory.
package memory

import (
	"encoding/binary"
	"net"
	"runtime"
	"sync"
	"time"

	yaml "gopkg.in/yaml.v2"

	"github.com/ProtocolONE/chihaya/bittorrent"
	"github.com/ProtocolONE/chihaya/pkg/log"
	"github.com/ProtocolONE/chihaya/pkg/stop"
	"github.com/ProtocolONE/chihaya/pkg/timecache"
	"github.com/ProtocolONE/chihaya/storage"
)

// Name is the name by which this peer store is registered with Chihaya.
const Name = "memory"

// Default config constants.
const (
	defaultShardCount                  = 1024
	defaultPrometheusReportingInterval = time.Second * 1
	defaultGarbageCollectionInterval   = time.Minute * 3
	defaultPeerLifetime                = time.Minute * 30
)

func init() {
	// Register the storage driver.
	storage.RegisterDriver(Name, driver{})
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
		"name":               Name,
		"gcInterval":         cfg.GarbageCollectionInterval,
		"promReportInterval": cfg.PrometheusReportingInterval,
		"peerLifetime":       cfg.PeerLifetime,
		"shardCount":         cfg.ShardCount,
	}
}

// Validate sanity checks values set in a config and returns a new config with
// default values replacing anything that is invalid.
//
// This function warns to the logger when a value is changed.
func (cfg Config) Validate() Config {
	validcfg := cfg

	if cfg.ShardCount <= 0 {
		validcfg.ShardCount = defaultShardCount
		log.Warn("falling back to default configuration", log.Fields{
			"name":     Name + ".ShardCount",
			"provided": cfg.ShardCount,
			"default":  validcfg.ShardCount,
		})
	}

	if cfg.GarbageCollectionInterval <= 0 {
		validcfg.GarbageCollectionInterval = defaultGarbageCollectionInterval
		log.Warn("falling back to default configuration", log.Fields{
			"name":     Name + ".GarbageCollectionInterval",
			"provided": cfg.GarbageCollectionInterval,
			"default":  validcfg.GarbageCollectionInterval,
		})
	}

	if cfg.PrometheusReportingInterval <= 0 {
		validcfg.PrometheusReportingInterval = defaultPrometheusReportingInterval
		log.Warn("falling back to default configuration", log.Fields{
			"name":     Name + ".PrometheusReportingInterval",
			"provided": cfg.PrometheusReportingInterval,
			"default":  validcfg.PrometheusReportingInterval,
		})
	}

	if cfg.PeerLifetime <= 0 {
		validcfg.PeerLifetime = defaultPeerLifetime
		log.Warn("falling back to default configuration", log.Fields{
			"name":     Name + ".PeerLifetime",
			"provided": cfg.PeerLifetime,
			"default":  validcfg.PeerLifetime,
		})
	}

	return validcfg
}

// New creates a new PeerStore backed by memory.
func New(provided Config) (storage.PeerStore, error) {
	cfg := provided.Validate()
	ps := &peerStore{
		cfg:    cfg,
		shards: make([]*peerShard, cfg.ShardCount*2),
		closed: make(chan struct{}),
	}

	for i := 0; i < cfg.ShardCount*2; i++ {
		ps.shards[i] = &peerShard{swarms: make(map[bittorrent.InfoHash]swarm)}
	}

	// Start a goroutine for garbage collection.
	ps.wg.Add(1)
	go func() {
		defer ps.wg.Done()
		for {
			select {
			case <-ps.closed:
				return
			case <-time.After(cfg.GarbageCollectionInterval):
				before := time.Now().Add(-cfg.PeerLifetime)
				log.Debug("storage: purging peers with no announces since", log.Fields{"before": before})
				ps.collectGarbage(before)
			}
		}
	}()

	// Start a goroutine for reporting statistics to Prometheus.
	ps.wg.Add(1)
	go func() {
		defer ps.wg.Done()
		t := time.NewTicker(cfg.PrometheusReportingInterval)
		for {
			select {
			case <-ps.closed:
				t.Stop()
				return
			case <-t.C:
				before := time.Now()
				ps.populateProm()
				log.Debug("storage: populateProm() finished", log.Fields{"timeTaken": time.Since(before)})
			}
		}
	}()

	return ps, nil
}

type serializedPeer string

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
	cfg    Config
	shards []*peerShard

	closed chan struct{}
	wg     sync.WaitGroup
}

var _ storage.PeerStore = &peerStore{}

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

	storage.PromInfohashesCount.Set(float64(numInfohashes))
	storage.PromSeedersCount.Set(float64(numSeeders))
	storage.PromLeechersCount.Set(float64(numLeechers))
}

// recordGCDuration records the duration of a GC sweep.
func recordGCDuration(duration time.Duration) {
	storage.PromGCDurationMilliseconds.Observe(float64(duration.Nanoseconds()) / float64(time.Millisecond))
}

func (ps *peerStore) getClock() int64 {
	return timecache.NowUnixNano()
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

func (ps *peerStore) PutSeeder(ih bittorrent.InfoHash, p bittorrent.Peer) error {
	select {
	case <-ps.closed:
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

	// If this peer isn't already a seeder, update the stats for the swarm.
	if _, ok := shard.swarms[ih].seeders[pk]; !ok {
		shard.numSeeders++
	}

	// Update the peer in the swarm.
	shard.swarms[ih].seeders[pk] = ps.getClock()

	shard.Unlock()
	return nil
}

func (ps *peerStore) DeleteSeeder(ih bittorrent.InfoHash, p bittorrent.Peer) error {
	select {
	case <-ps.closed:
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

	shard.numSeeders--
	delete(shard.swarms[ih].seeders, pk)

	if len(shard.swarms[ih].seeders)|len(shard.swarms[ih].leechers) == 0 {
		delete(shard.swarms, ih)
	}

	shard.Unlock()
	return nil
}

func (ps *peerStore) PutLeecher(ih bittorrent.InfoHash, p bittorrent.Peer) error {
	select {
	case <-ps.closed:
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

	// If this peer isn't already a leecher, update the stats for the swarm.
	if _, ok := shard.swarms[ih].leechers[pk]; !ok {
		shard.numLeechers++
	}

	// Update the peer in the swarm.
	shard.swarms[ih].leechers[pk] = ps.getClock()

	shard.Unlock()
	return nil
}

func (ps *peerStore) DeleteLeecher(ih bittorrent.InfoHash, p bittorrent.Peer) error {
	select {
	case <-ps.closed:
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

	shard.numLeechers--
	delete(shard.swarms[ih].leechers, pk)

	if len(shard.swarms[ih].seeders)|len(shard.swarms[ih].leechers) == 0 {
		delete(shard.swarms, ih)
	}

	shard.Unlock()
	return nil
}

func (ps *peerStore) GraduateLeecher(ih bittorrent.InfoHash, p bittorrent.Peer) error {
	select {
	case <-ps.closed:
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

	// If this peer is a leecher, update the stats for the swarm and remove them.
	if _, ok := shard.swarms[ih].leechers[pk]; ok {
		shard.numLeechers--
		delete(shard.swarms[ih].leechers, pk)
	}

	// If this peer isn't already a seeder, update the stats for the swarm.
	if _, ok := shard.swarms[ih].seeders[pk]; !ok {
		shard.numSeeders++
	}

	// Update the peer in the swarm.
	shard.swarms[ih].seeders[pk] = ps.getClock()

	shard.Unlock()
	return nil
}

func (ps *peerStore) AnnouncePeers(ih bittorrent.InfoHash, seeder bool, numWant int, announcer bittorrent.Peer) (peers []bittorrent.Peer, err error) {
	select {
	case <-ps.closed:
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
		for pk := range leechers {
			if numWant == 0 {
				break
			}

			peers = append(peers, decodePeerKey(pk))
			numWant--
		}
	} else {
		// Append as many seeders as possible.
		seeders := shard.swarms[ih].seeders
		for pk := range seeders {
			if numWant == 0 {
				break
			}

			peers = append(peers, decodePeerKey(pk))
			numWant--
		}

		// Append leechers until we reach numWant.
		if numWant > 0 {
			leechers := shard.swarms[ih].leechers
			announcerPK := newPeerKey(announcer)
			for pk := range leechers {
				if pk == announcerPK {
					continue
				}

				if numWant == 0 {
					break
				}

				peers = append(peers, decodePeerKey(pk))
				numWant--
			}
		}
	}

	shard.RUnlock()
	return
}

func (ps *peerStore) ScrapeSwarm(ih bittorrent.InfoHash, addressFamily bittorrent.AddressFamily) (resp bittorrent.Scrape) {
	select {
	case <-ps.closed:
		panic("attempted to interact with stopped memory store")
	default:
	}

	resp.InfoHash = ih
	shard := ps.shards[ps.shardIndex(ih, addressFamily)]
	shard.RLock()

	swarm, ok := shard.swarms[ih]
	if !ok {
		shard.RUnlock()
		return
	}

	resp.Incomplete = uint32(len(swarm.leechers))
	resp.Complete = uint32(len(swarm.seeders))
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
	case <-ps.closed:
		return nil
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
					shard.numLeechers--
					delete(shard.swarms[ih].leechers, pk)
				}
			}

			for pk, mtime := range shard.swarms[ih].seeders {
				if mtime <= cutoffUnix {
					shard.numSeeders--
					delete(shard.swarms[ih].seeders, pk)
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

func (ps *peerStore) Stop() stop.Result {
	c := make(stop.Channel)
	go func() {
		close(ps.closed)
		ps.wg.Wait()

		// Explicitly deallocate our storage.
		shards := make([]*peerShard, len(ps.shards))
		for i := 0; i < len(ps.shards); i++ {
			shards[i] = &peerShard{swarms: make(map[bittorrent.InfoHash]swarm)}
		}
		ps.shards = shards

		c.Done()
	}()

	return c.Result()
}

func (ps *peerStore) LogFields() log.Fields {
	return ps.cfg.LogFields()
}
