// Package redis implements the storage interface for a Chihaya
// BitTorrent tracker to keep peer data in redis.
//
// In general, each swarm is tracked in two hashes per address family:
//
// - s<20 byte infohash><one byte address family><one byte seeder/leecher marker> -> <peer key> -> <last modified time>
// Holds peers for the swarm identified by <infohash> with their last-active
// time.
// Garbage collection should periodically check the peers for expiration.
// They all have a s prefix so that we can SCAN over them.
//
// In addition to that, we use one hash per address family to track the number
// of swarms:
//
// - <address family string>_swarm_counts -> <20-byte infohash> counter
// Counts the number of peers per swarm, as the sum of seeders and leechers.
// This is used to get the total number of swarms.
//
// Additionally, two keys per address family are used to track the number of
// seeders and leechers separately.
//
// - <address family string>_seeder_count
//	Counts the number of seeders.
//
// - <address family string>_leecher_count
//	Counts the number of leechers.
package redis

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gomodule/redigo/redis"
	yaml "gopkg.in/yaml.v2"

	"github.com/chihaya/chihaya/bittorrent"
	"github.com/chihaya/chihaya/pkg/log"
	"github.com/chihaya/chihaya/pkg/stop"
	"github.com/chihaya/chihaya/pkg/timecache"
	"github.com/chihaya/chihaya/storage"
)

// Name is the name by which this peer store is registered with Chihaya.
const Name = "redis"

// Default config constants.
const (
	defaultPrometheusReportingInterval = time.Second * 1
	defaultGarbageCollectionInterval   = time.Minute * 3
	defaultPeerLifetime                = time.Minute * 30
	defaultRedisBroker                 = "redis://myRedis@127.0.0.1:6379/0"
	defaultRedisReadTimeout            = time.Second * 15
	defaultRedisWriteTimeout           = time.Second * 15
	defaultRedisConnectTimeout         = time.Second * 15
)

func init() {
	// Register the storage driver.
	storage.RegisterDriver(Name, driver{})
}

// ErrInternalError is returned if something fails within the driver that is
// not the fault of a client.
var ErrInternalError = errors.New("internal error")

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

// Config holds the configuration of a redis PeerStore.
type Config struct {
	GarbageCollectionInterval   time.Duration `yaml:"gc_interval"`
	PrometheusReportingInterval time.Duration `yaml:"prometheus_reporting_interval"`
	PeerLifetime                time.Duration `yaml:"peer_lifetime"`
	RedisBroker                 string        `yaml:"redis_broker"`
	RedisReadTimeout            time.Duration `yaml:"redis_read_timeout"`
	RedisWriteTimeout           time.Duration `yaml:"redis_write_timeout"`
	RedisConnectTimeout         time.Duration `yaml:"redis_connect_timeout"`
}

// LogFields renders the current config as a set of Logrus fields.
func (cfg Config) LogFields() log.Fields {
	return log.Fields{
		"name":                Name,
		"gcInterval":          cfg.GarbageCollectionInterval,
		"promReportInterval":  cfg.PrometheusReportingInterval,
		"peerLifetime":        cfg.PeerLifetime,
		"redisBroker":         cfg.RedisBroker,
		"redisReadTimeout":    cfg.RedisReadTimeout,
		"redisWriteTimeout":   cfg.RedisWriteTimeout,
		"redisConnectTimeout": cfg.RedisConnectTimeout,
	}
}

// Validate sanity checks values set in a config and returns a new config with
// default values replacing anything that is invalid.
//
// This function warns to the logger when a value is changed.
func (cfg Config) Validate() Config {
	validcfg := cfg

	if cfg.RedisBroker == "" {
		validcfg.RedisBroker = defaultRedisBroker
		log.Warn("falling back to default configuration", log.Fields{
			"name":     Name + ".RedisBroker",
			"provided": cfg.RedisBroker,
			"default":  validcfg.RedisBroker,
		})
	}

	if cfg.RedisReadTimeout <= 0 {
		validcfg.RedisReadTimeout = defaultRedisReadTimeout
		log.Warn("falling back to default configuration", log.Fields{
			"name":     Name + ".RedisReadTimeout",
			"provided": cfg.RedisReadTimeout,
			"default":  validcfg.RedisReadTimeout,
		})
	}

	if cfg.RedisWriteTimeout <= 0 {
		validcfg.RedisWriteTimeout = defaultRedisWriteTimeout
		log.Warn("falling back to default configuration", log.Fields{
			"name":     Name + ".RedisWriteTimeout",
			"provided": cfg.RedisWriteTimeout,
			"default":  validcfg.RedisWriteTimeout,
		})
	}

	if cfg.RedisConnectTimeout <= 0 {
		validcfg.RedisConnectTimeout = defaultRedisConnectTimeout
		log.Warn("falling back to default configuration", log.Fields{
			"name":     Name + ".RedisConnectTimeout",
			"provided": cfg.RedisConnectTimeout,
			"default":  validcfg.RedisConnectTimeout,
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

// New creates a new PeerStore backed by redis.
func New(provided Config) (storage.PeerStore, error) {
	cfg := provided.Validate()

	u, err := parseRedisURL(cfg.RedisBroker)
	if err != nil {
		return nil, err
	}

	ps := &peerStore{
		cfg: cfg,
		rb:  newRedisBackend(&provided, u, ""),
		scripts: scripts{
			putPeer:      newPutPeerScript(),
			deletePeer:   newDeletePeerScript(),
			graduatePeer: newGraduatePeerScript(),
			gcPeer:       newGCPeerScript(),
		},
		closed: make(chan struct{}),
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
				if err = ps.collectGarbage(before); err != nil {
					log.Error("storage: collectGarbage error", log.Fields{"before": before, "error": err})
				}
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

type peerKey string

func makePeerKey(p bittorrent.Peer) peerKey {
	b := make([]byte, 20+2+len(p.IP.IP))
	copy(b[:20], p.ID[:])
	binary.BigEndian.PutUint16(b[20:22], p.Port)
	copy(b[22:], p.IP.IP)

	return peerKey(b)
}

func decodePeerKey(pk peerKey) bittorrent.Peer {
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

type peerStore struct {
	cfg Config
	rb  *redisBackend

	scripts scripts

	closed chan struct{}
	wg     sync.WaitGroup
}

func addressFamilies() []bittorrent.AddressFamily {
	return []bittorrent.AddressFamily{bittorrent.IPv4, bittorrent.IPv6}
}

// Constants used for constructing/deconstructing swarm keys.
const (
	swarmPrefix               = "s"
	seedersSuffix             = byte(0)
	leechersSuffix            = byte(1)
	seederLeecherSuffixLength = 1
	// We encode address families as a single byte.
	addressFamilyLength              = 1
	swarmKeyLength                   = len(swarmPrefix) + 20 + addressFamilyLength + seederLeecherSuffixLength
	swarmKeyInfohashStart            = len(swarmPrefix)
	swarmKeyInfohashEnd              = swarmKeyInfohashStart + 20
	swarmKeyAddressFamilyIndex       = swarmKeyInfohashEnd
	swarmKeySeederLeecherSuffixIndex = swarmKeyAddressFamilyIndex + 1
)

// Counting key constants.
const (
	swarmCountsSuffix   = "_swarm_counts"
	seedersCountSuffix  = "_seeders_count"
	leechersCountSuffix = "_leechers_count"
)

// leecherSwarmKey derives a key for the leecher-side of the swarm for the
// given infohash and address family, in the form
// s<infohash bytes><address family byte><1 byte>
func leecherSwarmKey(addressFamily bittorrent.AddressFamily, infoHash bittorrent.InfoHash) string {
	var b strings.Builder
	b.Grow(swarmKeyLength)

	b.WriteString(swarmPrefix)
	b.Write(infoHash[:])
	b.WriteByte(byte(addressFamily))
	b.WriteByte(leechersSuffix)

	return b.String()
}

// seederSwarmKey derives a key for the seeder-side of the swarm for the
// given infohash and address family, in the form
// s<infohash bytes><address family byte><0 byte>.
func seederSwarmKey(addressFamily bittorrent.AddressFamily, infoHash bittorrent.InfoHash) string {
	var b strings.Builder
	b.Grow(swarmKeyLength)

	b.WriteString(swarmPrefix)
	b.Write(infoHash[:])
	b.WriteByte(byte(addressFamily))
	b.WriteByte(seedersSuffix)

	return b.String()
}

func deconstructSwarmKey(swarmKey string) (bittorrent.InfoHash, bittorrent.AddressFamily, bool) {
	if len(swarmKey) < swarmKeyLength {
		// don't print the key as a string, it probably contains a binary
		// infohash
		panic(fmt.Sprintf("invalid swarmKey: %#v", swarmKey[:]))
	}

	infohashPart := swarmKey[swarmKeyInfohashStart:swarmKeyInfohashEnd]
	addressFamilyByte := swarmKey[swarmKeyAddressFamilyIndex]
	seeder := swarmKey[swarmKeySeederLeecherSuffixIndex] == seedersSuffix

	return bittorrent.InfoHashFromString(infohashPart), bittorrent.AddressFamily(addressFamilyByte), seeder
}

func addressFamiliedKey(addressFamily bittorrent.AddressFamily, suffix string) string {
	var b strings.Builder
	addressFamilyString := addressFamily.String()

	b.Grow(len(addressFamilyString) + len(suffix))
	b.WriteString(addressFamilyString)
	b.WriteString(suffix)

	return b.String()
}

func swarmCountsKey(addressFamily bittorrent.AddressFamily) string {
	return addressFamiliedKey(addressFamily, swarmCountsSuffix)
}

func seederCountKey(addressFamily bittorrent.AddressFamily) string {
	return addressFamiliedKey(addressFamily, seedersCountSuffix)
}

func leecherCountKey(addressFamily bittorrent.AddressFamily) string {
	return addressFamiliedKey(addressFamily, leechersCountSuffix)
}

// populateProm aggregates metrics over all groups and then posts them to
// prometheus.
func (ps *peerStore) populateProm() {
	var numInfohashes, numSeeders, numLeechers int64

	conn := ps.rb.open()
	defer conn.Close()

	for _, addressFamily := range addressFamilies() {
		if n, err := redis.Int64(conn.Do("HLEN", swarmCountsKey(addressFamily))); err != nil && err != redis.ErrNil {
			log.Error("storage: HLEN failed", log.Fields{
				"key":   swarmCountsKey(addressFamily),
				"error": err,
			})
		} else {
			numInfohashes += n
		}
		if n, err := redis.Int64(conn.Do("GET", seederCountKey(addressFamily))); err != nil && err != redis.ErrNil {
			log.Error("storage: GET failed", log.Fields{
				"key":   seederCountKey(addressFamily),
				"error": err,
			})
		} else {
			numSeeders += n
		}
		if n, err := redis.Int64(conn.Do("GET", leecherCountKey(addressFamily))); err != nil && err != redis.ErrNil {
			log.Error("storage: GET failed", log.Fields{
				"key":   leecherCountKey(addressFamily),
				"error": err,
			})
		} else {
			numLeechers += n
		}
	}

	storage.PromInfohashesCount.Set(float64(numInfohashes))
	storage.PromSeedersCount.Set(float64(numSeeders))
	storage.PromLeechersCount.Set(float64(numLeechers))
}

func (ps *peerStore) getClock() int64 {
	return timecache.NowUnixNano()
}

func (ps *peerStore) PutSeeder(ih bittorrent.InfoHash, p bittorrent.Peer) error {
	log.Debug("storage: PutSeeder", log.Fields{
		"InfoHash": ih.String(),
		"Peer":     p,
	})

	select {
	case <-ps.closed:
		panic("attempted to interact with stopped redis store")
	default:
	}

	conn := ps.rb.open()
	defer conn.Close()

	err := ps.scripts.putPeer.execute(conn, p, ps.getClock(), ih, true)
	if err != nil {
		log.Error("storage: putPeer script execution failed", log.Fields{
			"error": err,
		})
		return ErrInternalError
	}

	return nil
}

func (ps *peerStore) DeleteSeeder(ih bittorrent.InfoHash, p bittorrent.Peer) error {
	log.Debug("storage: DeleteSeeder", log.Fields{
		"InfoHash": ih.String(),
		"Peer":     p,
	})

	select {
	case <-ps.closed:
		panic("attempted to interact with stopped redis store")
	default:
	}

	conn := ps.rb.open()
	defer conn.Close()

	numDeleted, err := ps.scripts.deletePeer.execute(conn, p, ih, true)
	if err != nil {
		log.Error("storage: deletePeer script execution failed", log.Fields{
			"error": err,
		})
		return ErrInternalError
	}
	if numDeleted == 0 {
		return storage.ErrResourceDoesNotExist
	}

	return nil
}

func (ps *peerStore) PutLeecher(ih bittorrent.InfoHash, p bittorrent.Peer) error {
	log.Debug("storage: PutLeecher", log.Fields{
		"InfoHash": ih.String(),
		"Peer":     p,
	})

	select {
	case <-ps.closed:
		panic("attempted to interact with stopped redis store")
	default:
	}

	conn := ps.rb.open()
	defer conn.Close()

	err := ps.scripts.putPeer.execute(conn, p, ps.getClock(), ih, false)
	if err != nil {
		log.Error("storage: putPeer script execution failed", log.Fields{
			"error": err,
		})
		return ErrInternalError
	}
	return nil
}

func (ps *peerStore) DeleteLeecher(ih bittorrent.InfoHash, p bittorrent.Peer) error {
	log.Debug("storage: DeleteLeecher", log.Fields{
		"InfoHash": ih.String(),
		"Peer":     p,
	})

	select {
	case <-ps.closed:
		panic("attempted to interact with stopped redis store")
	default:
	}

	conn := ps.rb.open()
	defer conn.Close()

	numDeleted, err := ps.scripts.deletePeer.execute(conn, p, ih, false)
	if err != nil {
		log.Error("storage: deletePeer script execution failed", log.Fields{
			"error": err,
		})
		return ErrInternalError
	}
	if numDeleted == 0 {
		return storage.ErrResourceDoesNotExist
	}

	return nil
}

func (ps *peerStore) GraduateLeecher(ih bittorrent.InfoHash, p bittorrent.Peer) error {
	log.Debug("storage: GraduateLeecher", log.Fields{
		"InfoHash": ih.String(),
		"Peer":     p,
	})

	select {
	case <-ps.closed:
		panic("attempted to interact with stopped redis store")
	default:
	}

	conn := ps.rb.open()
	defer conn.Close()

	err := ps.scripts.graduatePeer.execute(conn, p, ps.getClock(), ih)
	if err != nil {
		log.Error("storage: graduatePeer script execution failed", log.Fields{
			"error": err,
		})
		return ErrInternalError
	}
	return nil
}

func (ps *peerStore) AnnouncePeers(ih bittorrent.InfoHash, seeder bool, numWant int, announcer bittorrent.Peer) (peers []bittorrent.Peer, err error) {
	log.Debug("storage: AnnouncePeers", log.Fields{
		"InfoHash": ih.String(),
		"seeder":   seeder,
		"numWant":  numWant,
		"Peer":     announcer,
	})

	select {
	case <-ps.closed:
		panic("attempted to interact with stopped redis store")
	default:
	}

	encodedLeecherInfoHash := leecherSwarmKey(announcer.IP.AddressFamily, ih)
	encodedSeederInfoHash := seederSwarmKey(announcer.IP.AddressFamily, ih)

	conn := ps.rb.open()
	defer conn.Close()

	leechers, err := conn.Do("HKEYS", encodedLeecherInfoHash)
	if err != nil {
		log.Error("storage: HKEYS failed", log.Fields{
			"key":   encodedLeecherInfoHash,
			"error": err,
		})
		return nil, ErrInternalError
	}
	conLeechers := leechers.([]interface{})

	seeders, err := conn.Do("HKEYS", encodedSeederInfoHash)
	if err != nil {
		log.Error("storage: HKEYS failed", log.Fields{
			"key":   encodedSeederInfoHash,
			"error": err,
		})
		return nil, ErrInternalError
	}
	conSeeders := seeders.([]interface{})

	if len(conLeechers) == 0 && len(conSeeders) == 0 {
		return nil, storage.ErrResourceDoesNotExist
	}

	if seeder {
		// Append leechers as possible.
		for _, pk := range conLeechers {
			if numWant == 0 {
				break
			}

			peers = append(peers, decodePeerKey(peerKey(pk.([]byte))))
			numWant--
		}
	} else {
		// Append as many seeders as possible.
		for _, pk := range conSeeders {
			if numWant == 0 {
				break
			}

			peers = append(peers, decodePeerKey(peerKey(pk.([]byte))))
			numWant--
		}

		// Append leechers until we reach numWant.
		if numWant > 0 {
			announcerPK := makePeerKey(announcer)
			for _, pk := range conLeechers {
				if pk == announcerPK {
					continue
				}

				if numWant == 0 {
					break
				}

				peers = append(peers, decodePeerKey(peerKey(pk.([]byte))))
				numWant--
			}
		}
	}

	return
}

func (ps *peerStore) ScrapeSwarm(ih bittorrent.InfoHash, af bittorrent.AddressFamily) (resp bittorrent.Scrape) {
	select {
	case <-ps.closed:
		panic("attempted to interact with stopped redis store")
	default:
	}

	resp.InfoHash = ih
	encodedLeecherInfoHash := leecherSwarmKey(af, ih)
	encodedSeederInfoHash := seederSwarmKey(af, ih)

	conn := ps.rb.open()
	defer conn.Close()

	leechersLen, err := redis.Int64(conn.Do("HLEN", encodedLeecherInfoHash))
	if err != nil {
		log.Error("storage: HLEN failed", log.Fields{
			"key":   encodedLeecherInfoHash,
			"error": err,
		})
		return
	}

	seedersLen, err := redis.Int64(conn.Do("HLEN", encodedSeederInfoHash))
	if err != nil {
		log.Error("storage: HLEN failed", log.Fields{
			"key":   encodedSeederInfoHash,
			"error": err,
		})
		return
	}

	resp.Incomplete = uint32(leechersLen)
	resp.Complete = uint32(seedersLen)

	return
}

func (ps *peerStore) collectGarbageInner(conn redis.Conn, cutoffUnix int64, swarmKeys []string) error {
	for _, swarmKey := range swarmKeys {
		// TODO doing this gives us safety, but might cost performance?
		infoHash, _, isSeeder := deconstructSwarmKey(swarmKey)

		// list all (peer, modified time) pairs for the infohash
		// TODO use SCAN for this too?
		peerList, err := redis.Strings(conn.Do("HGETALL", swarmKey))
		if err != nil {
			log.Error("storage: HGETALL failed", log.Fields{
				"key":   swarmKey,
				"error": err,
			})
			return err
		}

		var pk peerKey
		for index, field := range peerList {
			// Redis sends back a zipped list with alternating keys and values.
			if index%2 == 0 { // Even indices are keys.
				pk = peerKey([]byte(field))
			} else { // Odd indices are values.
				mtime, err := strconv.ParseInt(field, 10, 64)
				if err != nil {
					return err
				}
				if mtime <= cutoffUnix {
					peer := decodePeerKey(pk)
					log.Debug("storage: deleting peer", log.Fields{
						"infoHash": infoHash.String(),
						"isSeeder": isSeeder,
						"peer":     peer.String(),
					})

					_, err := ps.scripts.gcPeer.execute(conn, peer, mtime, infoHash, isSeeder)
					if err != nil {
						log.Error("storage: gcPeer script execution failed", log.Fields{
							"error": err,
						})
						return err
					}

				}
			}
		}
	}

	return nil
}

// collectGarbage deletes all Peers from the PeerStore which are older than the
// cutoff time.
//
// This function must be able to execute while other methods on this interface
// are being executed in parallel.
//
// It uses atomic scripts and a CAS logic to do that.
func (ps *peerStore) collectGarbage(cutoff time.Time) error {
	select {
	case <-ps.closed:
		return nil
	default:
	}

	conn := ps.rb.open()
	defer conn.Close()

	cutoffUnix := cutoff.UnixNano()
	start := time.Now()

	var (
		cursor int64
		items  []string
	)

	// SCAN through all of our swarm keys.
	for {
		values, err := redis.Values(conn.Do("SCAN", cursor, "MATCH", fmt.Sprintf("%s*", swarmPrefix), "COUNT", 50))
		if err != nil {
			log.Error("storage: SCAN failed", log.Fields{
				"error": err,
			})
			return err
		}

		values, err = redis.Scan(values, &cursor, &items)
		if err != nil {
			log.Error("storage: unable to parse redis response", log.Fields{
				"error": err,
			})
			return err
		}

		err = ps.collectGarbageInner(conn, cutoffUnix, items)
		if err != nil {
			log.Error("storage: collectGarbageInner failed", log.Fields{
				"error": err,
			})
			return err
		}

		if cursor == 0 {
			break
		}
	}

	duration := float64(time.Since(start).Nanoseconds()) / float64(time.Millisecond)
	log.Debug("storage: recordGCDuration", log.Fields{"timeTaken(ms)": duration})
	storage.PromGCDurationMilliseconds.Observe(duration)

	return nil
}

func (ps *peerStore) Stop() stop.Result {
	c := make(stop.Channel)
	go func() {
		close(ps.closed)
		ps.wg.Wait()
		log.Info("storage: exiting. Chihaya does not clear data in redis when exiting. Consult the Chihaya documentation for information about redis keys.")
		c.Done()
	}()

	return c.Result()
}

func (ps *peerStore) LogFields() log.Fields {
	return ps.cfg.LogFields()
}
