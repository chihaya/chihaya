// Package redis implements the storage interface for a Chihaya
// BitTorrent tracker keeping peer data in redis.
package redis

import (
	"encoding/binary"
	"net"
	neturl "net/url"
	"sync"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/chihaya/chihaya/bittorrent"
	"github.com/chihaya/chihaya/pkg/log"
	"github.com/chihaya/chihaya/pkg/timecache"
	"github.com/chihaya/chihaya/storage"

	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/chihaya/chihaya/storage/redis/common"
	"github.com/garyburd/redigo/redis"
	"gopkg.in/redsync.v1"
)

// Name is the name by which this peer store is registered with Chihaya.
const Name = "redis"

// Default config constants.
const (
	defaultPrometheusReportingInterval = time.Second * 1
	defaultGarbageCollectionInterval   = time.Minute * 3
	defaultPeerLifetime                = time.Minute * 30
	defaultRedisBroker                 = "redis://myRedis@127.0.0.1:6379/0"
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

// Config holds the configuration of a redis PeerStore.
type Config struct {
	GarbageCollectionInterval   time.Duration `yaml:"gc_interval"`
	PrometheusReportingInterval time.Duration `yaml:"prometheus_reporting_interval"`
	PeerLifetime                time.Duration `yaml:"peer_lifetime"`
	RedisBroker                 string        `yaml:"redis_broker"`
}

// LogFields renders the current config as a set of Logrus fields.
func (cfg Config) LogFields() log.Fields {
	return log.Fields{
		"name":               Name,
		"gcInterval":         cfg.GarbageCollectionInterval,
		"promReportInterval": cfg.PrometheusReportingInterval,
		"peerLifetime":       cfg.PeerLifetime,
		"redisBroker":        cfg.RedisBroker,
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

// ParseRedisURL ...
func ParseRedisURL(url string) (host, password string, db int, err error) {
	// redis://pwd@host/db

	var u *neturl.URL
	u, err = neturl.Parse(url)
	if err != nil {
		return
	}
	if u.Scheme != "redis" {
		err = errors.New("No redis scheme found")
		return
	}

	if u.User != nil {
		password = u.User.String()
	}

	host = u.Host

	parts := strings.Split(u.Path, "/")
	if len(parts) == 1 {
		db = 0 //default redis db
	} else {
		db, err = strconv.Atoi(parts[1])
		if err != nil {
			db, err = 0, nil //ignore err here
		}
	}

	return
}

// NewRedisBackend creates RedisBackend instance
func NewRedisBackend(host, password, socketPath string, db int) *RedisBackend {
	return &RedisBackend{
		host:       host,
		db:         db,
		password:   password,
		socketPath: socketPath,
	}
}

// open returns or creates instance of Redis connection
func (rb *RedisBackend) open() redis.Conn {
	if rb.pool == nil {
		rb.pool = rb.NewPool(rb.socketPath, rb.host, rb.password, rb.db)
	}
	if rb.redsync == nil {
		var pools = []redsync.Pool{rb.pool}
		rb.redsync = redsync.New(pools)
	}
	return rb.pool.Get()
}

// New creates a new PeerStore backed by redis.
func New(provided Config) (storage.PeerStore, error) {
	cfg := provided.Validate()

	// creates RedisBackend instance
	h, p, db, err := ParseRedisURL(cfg.RedisBroker)
	if err != nil {
		return nil, err
	}

	ps := &peerStore{
		cfg:    cfg,
		rb:     NewRedisBackend(h, p, "", db),
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

// RedisBackend represents a Memcache result backend
type RedisBackend struct {
	host     string
	password string
	db       int
	pool     *redis.Pool
	// If set, path to a socket file overrides hostname
	socketPath string
	redsync    *redsync.Redsync
	common.RedisConnector
}

type peerStore struct {
	cfg Config
	rb  *RedisBackend

	closed chan struct{}
	wg     sync.WaitGroup
}

// populateProm aggregates metrics over all shards and then posts them to
// prometheus.
func (ps *peerStore) populateProm() {
	var numInfohashes, numSeeders, numLeechers uint64

	shards := [2]string{bittorrent.IPv4.String(), bittorrent.IPv6.String()}

	conn := ps.rb.open()
	defer conn.Close()

	for _, shard := range shards {
		infohashes_list, err := conn.Do("HKEYS", shard) // key
		if err != nil {
			return
		}
		infohashes := infohashes_list.([]interface{})

		InfohashLPrefix := shard + "_L_"
		InfohashSPrefix := shard + "_S_"
		InfohashPrefixLen := len(InfohashLPrefix)
		InfohashesMap := make(map[string]bool)

		for _, ih := range infohashes {
			ih_str := string(ih.([]byte))
			ih_str_infohash := ih_str[InfohashPrefixLen:]
			if strings.HasPrefix(ih_str, InfohashLPrefix) {
				numLeechers++
				InfohashesMap[ih_str_infohash] = true
			} else if strings.HasPrefix(ih_str, InfohashSPrefix) {
				numSeeders++
				InfohashesMap[ih_str_infohash] = true
			} else {
				log.Error("storage: invalid Redis state", log.Fields{
					"Hkey":   shard,
					"Hfield": ih_str,
				})
			}
		}

		numInfohashes += uint64(len(InfohashesMap))
	}

	storage.PromInfohashesCount.Set(float64(numInfohashes))
	storage.PromSeedersCount.Set(float64(numSeeders))
	storage.PromLeechersCount.Set(float64(numLeechers))

	log.Debug("storage: populateProm() aggregates metrics over all shards", log.Fields{
		"numInfohashes": float64(numInfohashes),
		"numSeeders":    float64(numSeeders),
		"numLeechers":   float64(numLeechers),
	})
}

// recordGCDuration records the duration of a GC sweep.
func recordGCDuration(duration time.Duration) {
	log.Debug("storage: recordGCDuration", log.Fields{"timeTaken(ms)": float64(duration.Nanoseconds()) / float64(time.Millisecond)})
	storage.PromGCDurationMilliseconds.Observe(float64(duration.Nanoseconds()) / float64(time.Millisecond))
}

func (ps *peerStore) getClock() int64 {
	return timecache.NowUnixNano()
}

func (ps *peerStore) PutSeeder(ih bittorrent.InfoHash, p bittorrent.Peer) error {
	IPver := p.IP.AddressFamily.String()
	log.Debug("storage: PutSeeder", log.Fields{
		"InfoHash": ih.String(),
		"Peer":     fmt.Sprintf("[ID: %s, IP: %s(AddressFamily: %s), Port %d]", p.ID.String(), p.IP.String(), IPver, p.Port),
	})

	select {
	case <-ps.closed:
		panic("attempted to interact with stopped redis store")
	default:
	}

	pk := newPeerKey(p)

	conn := ps.rb.open()
	defer conn.Close()

	// Update the peer in the swarm.
	encodedSeederInfoHash := IPver + "_S_" + ih.String()
	ct := ps.getClock()
	_, err := conn.Do("HSET", encodedSeederInfoHash, pk, ct)
	if err != nil {
		return err
	}
	_, err = conn.Do("HSET", IPver, encodedSeederInfoHash, ct)
	if err != nil {
		return err
	}

	return nil
}

func (ps *peerStore) DeleteSeeder(ih bittorrent.InfoHash, p bittorrent.Peer) error {
	IPver := p.IP.AddressFamily.String()
	log.Debug("storage: DeleteSeeder", log.Fields{
		"InfoHash": ih.String(),
		"Peer":     fmt.Sprintf("[ID: %s, IP: %s(AddressFamily: %s), Port %d]", p.ID.String(), p.IP.String(), IPver, p.Port),
	})

	select {
	case <-ps.closed:
		panic("attempted to interact with stopped redis store")
	default:
	}

	pk := newPeerKey(p)

	conn := ps.rb.open()
	defer conn.Close()

	encodedSeederInfoHash := IPver + "_S_" + ih.String()

	DelNum, err := conn.Do("HDEL", encodedSeederInfoHash, pk)
	if err != nil {
		return err
	}
	if DelNum.(int64) == 0 {
		return storage.ErrResourceDoesNotExist
	}

	return nil
}

func (ps *peerStore) PutLeecher(ih bittorrent.InfoHash, p bittorrent.Peer) error {
	IPver := p.IP.AddressFamily.String()
	log.Debug("storage: PutLeecher", log.Fields{
		"InfoHash": ih.String(),
		"Peer":     fmt.Sprintf("[ID: %s, IP: %s(AddressFamily: %s), Port %d]", p.ID.String(), p.IP.String(), IPver, p.Port),
	})

	select {
	case <-ps.closed:
		panic("attempted to interact with stopped redis store")
	default:
	}

	pk := newPeerKey(p)

	conn := ps.rb.open()
	defer conn.Close()

	// Update the peer in the swarm.
	encodedLeecherInfoHash := IPver + "_L_" + ih.String()
	ct := ps.getClock()
	_, err := conn.Do("HSET", encodedLeecherInfoHash, pk, ct)
	if err != nil {
		return err
	}
	_, err = conn.Do("HSET", IPver, encodedLeecherInfoHash, ct)
	if err != nil {
		return err
	}

	return nil
}

func (ps *peerStore) DeleteLeecher(ih bittorrent.InfoHash, p bittorrent.Peer) error {
	IPver := p.IP.AddressFamily.String()
	log.Debug("storage: DeleteLeecher", log.Fields{
		"InfoHash": ih.String(),
		"Peer":     fmt.Sprintf("[ID: %s, IP: %s(AddressFamily: %s), Port %d]", p.ID.String(), p.IP.String(), IPver, p.Port),
	})

	select {
	case <-ps.closed:
		panic("attempted to interact with stopped redis store")
	default:
	}

	pk := newPeerKey(p)

	conn := ps.rb.open()
	defer conn.Close()

	encodedLeecherInfoHash := IPver + "_L_" + ih.String()

	DelNum, err := conn.Do("HDEL", encodedLeecherInfoHash, pk)
	if err != nil {
		return err
	}
	if DelNum.(int64) == 0 {
		return storage.ErrResourceDoesNotExist
	}

	return nil
}

func (ps *peerStore) GraduateLeecher(ih bittorrent.InfoHash, p bittorrent.Peer) error {
	IPver := p.IP.AddressFamily.String()
	log.Debug("storage: GraduateLeecher", log.Fields{
		"InfoHash": ih.String(),
		"Peer":     fmt.Sprintf("[ID: %s, IP: %s(AddressFamily: %s), Port %d]", p.ID.String(), p.IP.String(), IPver, p.Port),
	})

	select {
	case <-ps.closed:
		panic("attempted to interact with stopped redis store")
	default:
	}

	pk := newPeerKey(p)

	conn := ps.rb.open()
	defer conn.Close()

	encodedInfoHash := ih.String()
	encodedLeecherInfoHash := IPver + "_L_" + encodedInfoHash
	encodedSeederInfoHash := IPver + "_S_" + encodedInfoHash

	_, err := conn.Do("HDEL", encodedLeecherInfoHash, pk)
	if err != nil {
		return err
	}

	// Update the peer in the swarm.
	ct := ps.getClock()
	_, err = conn.Do("HSET", encodedSeederInfoHash, pk, ct)
	if err != nil {
		return err
	}
	_, err = conn.Do("HSET", IPver, encodedSeederInfoHash, ct)
	if err != nil {
		return err
	}

	return nil
}

func (ps *peerStore) AnnouncePeers(ih bittorrent.InfoHash, seeder bool, numWant int, announcer bittorrent.Peer) (peers []bittorrent.Peer, err error) {
	IPver := announcer.IP.AddressFamily.String()
	log.Debug("storage: AnnouncePeers", log.Fields{
		"InfoHash": ih.String(),
		"seeder":   seeder,
		"numWant":  numWant,
		"Peer":     fmt.Sprintf("[ID: %s, IP: %s(AddressFamily: %s), Port %d]", announcer.ID.String(), announcer.IP.String(), IPver, announcer.Port),
	})

	select {
	case <-ps.closed:
		panic("attempted to interact with stopped redis store")
	default:
	}

	encodedInfoHash := ih.String()
	encodedLeecherInfoHash := IPver + "_L_" + encodedInfoHash // key
	encodedSeederInfoHash := IPver + "_S_" + encodedInfoHash  // key

	conn := ps.rb.open()
	defer conn.Close()

	leechers, err := conn.Do("HKEYS", encodedLeecherInfoHash)
	if err != nil {
		return nil, err
	}
	conLeechers := leechers.([]interface{})

	seeders, err := conn.Do("HKEYS", encodedSeederInfoHash)
	if err != nil {
		return nil, err
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

			peers = append(peers, decodePeerKey(serializedPeer(pk.([]byte))))
			numWant--
		}
	} else {
		// Append as many seeders as possible.
		for _, pk := range conSeeders {
			if numWant == 0 {
				break
			}

			peers = append(peers, decodePeerKey(serializedPeer(pk.([]byte))))
			numWant--
		}

		// Append leechers until we reach numWant.
		if numWant > 0 {
			announcerPK := newPeerKey(announcer)
			for _, pk := range conLeechers {
				if pk == announcerPK {
					continue
				}

				if numWant == 0 {
					break
				}

				peers = append(peers, decodePeerKey(serializedPeer(pk.([]byte))))
				numWant--
			}
		}
	}

	APResult := ""
	for _, pr := range peers {
		APResult = fmt.Sprintf("%s Peer:[ID: %s, IP: %s(AddressFamily: %s), Port %d]", APResult, pr.ID.String(), pr.IP.String(), IPver, pr.Port)
	}
	log.Debug("storage: AnnouncePeers result", log.Fields{
		"peers": APResult,
	})

	return
}

func (ps *peerStore) ScrapeSwarm(ih bittorrent.InfoHash, addressFamily bittorrent.AddressFamily) (resp bittorrent.Scrape) {
	select {
	case <-ps.closed:
		panic("attempted to interact with stopped redis store")
	default:
	}

	resp.InfoHash = ih
	IPver := addressFamily.String()
	encodedInfoHash := ih.String()
	encodedLeecherInfoHash := IPver + "_L_" + encodedInfoHash // key
	encodedSeederInfoHash := IPver + "_S_" + encodedInfoHash  // key

	conn := ps.rb.open()
	defer conn.Close()

	leechersLen, err := conn.Do("HLEN", encodedLeecherInfoHash)
	if err != nil {
		log.Error("storage: Redis HLEN failure", log.Fields{
			"Hkey":  encodedLeecherInfoHash,
			"error": err,
		})
		return
	}
	lLen := leechersLen.(int64)

	seedersLen, err := conn.Do("HLEN", encodedSeederInfoHash)
	if err != nil {
		log.Error("storage: Redis HLEN failure", log.Fields{
			"Hkey":  encodedSeederInfoHash,
			"error": err,
		})
		return
	}
	sLen := seedersLen.(int64)

	if lLen == 0 && sLen == 0 {
		return
	}

	resp.Incomplete = uint32(lLen)
	resp.Complete = uint32(sLen)

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

	shards := [2]string{bittorrent.IPv4.String(), bittorrent.IPv6.String()}

	conn := ps.rb.open()
	defer conn.Close()

	cutoffUnix := cutoff.UnixNano()
	start := time.Now()

	for _, shard := range shards {
		infohashesList, err := conn.Do("HKEYS", shard) // key
		if err != nil {
			return err
		}
		infohashes := infohashesList.([]interface{})

		for _, ih := range infohashes {
			ihStr := string(ih.([]byte))

			ihList, err := conn.Do("HGETALL", ihStr) // field
			if err != nil {
				return err
			}
			conIhList := ihList.([]interface{})

			if len(conIhList) == 0 {
				_, err := conn.Do("DEL", ihStr)
				if err != nil {
					return err
				}
				log.Debug("storage: Deleting Redis", log.Fields{"Hkey": ihStr})
				_, err = conn.Do("HDEL", shard, ihStr)
				if err != nil {
					return err
				}
				log.Debug("storage: Deleting Redis", log.Fields{
					"Hkey":   shard,
					"Hfield": ihStr,
				})
				continue
			}

			var pk serializedPeer
			for index, ihField := range conIhList {
				if index%2 != 0 { // value
					mtime, err := strconv.ParseInt(string(ihField.([]byte)), 10, 64)
					if err != nil {
						return err
					}
					if mtime <= cutoffUnix {
						_, err := conn.Do("HDEL", ihStr, pk)
						if err != nil {
							return err
						}
						p := decodePeerKey(pk)
						log.Debug("storage: Deleting peer", log.Fields{
							"Peer": fmt.Sprintf("[ID: %s, IP: %s(AddressFamily: %s), Port %d]", p.ID.String(), p.IP.String(), p.IP.AddressFamily.String(), p.Port),
						})
					}
				} else { // key
					pk = serializedPeer(ihField.([]byte))
				}
			}

			ihLen, err := conn.Do("HLEN", ihStr)
			if err != nil {
				return err
			}
			if ihLen.(int64) == 0 {
				_, err := conn.Do("DEL", ihStr)
				if err != nil {
					return err
				}
				log.Debug("storage: Deleting Redis", log.Fields{"Hkey": ihStr})
				_, err = conn.Do("HDEL", shard, ihStr)
				if err != nil {
					return err
				}
				log.Debug("storage: Deleting Redis", log.Fields{
					"Hkey":   shard,
					"Hfield": ihStr,
				})
			}

		}

	}

	recordGCDuration(time.Since(start))

	return nil
}

func (ps *peerStore) Stop() <-chan error {
	c := make(chan error)
	go func() {
		close(ps.closed)
		ps.wg.Wait()

		// TODO(duyanghao): something to be done?

		close(c)
	}()

	return c
}

func (ps *peerStore) LogFields() log.Fields {
	return ps.cfg.LogFields()
}
