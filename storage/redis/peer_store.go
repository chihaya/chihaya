// Package redis implements the storage interface for a Chihaya
// BitTorrent tracker keeping peer data in redis with hash.
//
// Hash keys are used in two different ways:
// - Swarm Keys
//   "(4|6)(L|S)(<infohash>)"
//   Stores peers in the swarm for the infohash.
//   Primarily used for responding to requests.
// - Infohash Keys
//   "(4|6)"
//   Stores the infohashes for which a swarm exists.
//   Primarily used for garbage collection and metrics.
//
// Tree keys are used to record the count of swarms and peers:
// - Infohash Count Key
//   "I(4|6)"
//   Stores the number of infohashes grouped by IP protocol.
// - Peer Count Keys
//   (4|6)(L|S)
//   Stores the number of peers grouped by IP protocol and download status.
package redis

import (
	"errors"
	"net/netip"
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
		cfg:    cfg,
		rb:     newRedisBackend(&provided, u, ""),
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

type peerStore struct {
	cfg Config
	rb  *redisBackend

	closed chan struct{}
	wg     sync.WaitGroup
}

var addressFamilies = [2]string{"4", "6"}

// Key that stores the peers for an InfoHash.
//
// "4S0102030405060708090a0b0c0d0e0f1011121314" => IPv4 Seeder with InfoHash 0102030405060708090a0b0c0d0e0f1011121314
// "6L0204090405060708060a0b0c0d0e0f1011121328" => IPv6 Leecher with Infohash 0204090405060708060a0b0c0d0e0f1011121328
func swarmKey(infoHash bittorrent.InfoHash, seed bool, addr netip.Addr) string {
	var b strings.Builder
	b.Grow(1 + 1 + 20) // "4"/"6" + "S"/"L" + len(InfoHash)

	if addr.Is4() {
		b.WriteString("4")
	} else {
		b.WriteString("6")
	}

	if seed {
		b.WriteString("S")
	} else {
		b.WriteString("L")
	}

	b.Write(infoHash.MarshalBinary())
	return b.String()
}

// Key that stores the cardinality of the peers of an IP version.
//
// "4L" => IPv4 Leechers
// "6S" => IPv6 Seeders
func peerCountKey(seed bool, addr netip.Addr) string {
	key := "4"
	if addr.Is6() {
		key = "6"
	}

	if seed {
		return key + "S"
	}
	return key + "L"
}

// Key that stores the total number of infohashes that has peers of an IP
// version.
//
// "I4" => number of IPv4 InfoHashes
// "I6" => number of IPv6 InfoHashes
func infohashCountKey(addr netip.Addr) string {
	if addr.Is4() {
		return "I4"
	}
	return "I6"
}

// populateProm aggregates metrics over all groups and then posts them to
// prometheus.
func (ps *peerStore) populateProm() {
	var numInfohashes, numSeeders, numLeechers int64

	conn := ps.rb.open()
	defer conn.Close()

	for _, af := range addressFamilies {
		infohashCountKey := "I" + af
		if n, err := redis.Int64(conn.Do("GET", infohashCountKey)); err != nil && !errors.Is(err, redis.ErrNil) {
			log.Error("storage: GET counter failure", log.Fields{
				"key":   infohashCountKey,
				"error": err,
			})
		} else {
			numInfohashes += n
		}

		seederCountKey := af + "S"
		if n, err := redis.Int64(conn.Do("GET", seederCountKey)); err != nil && !errors.Is(err, redis.ErrNil) {
			log.Error("storage: GET counter failure", log.Fields{
				"key":   seederCountKey,
				"error": err,
			})
		} else {
			numSeeders += n
		}

		leecherCountKey := af + "L"
		if n, err := redis.Int64(conn.Do("GET", leecherCountKey)); err != nil && !errors.Is(err, redis.ErrNil) {
			log.Error("storage: GET counter failure", log.Fields{
				"key":   leecherCountKey,
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

	addr := p.AddrPort.Addr()
	sk := swarmKey(ih, true, addr)
	ihKey := sk[0]
	ct := timecache.NowUnixNano()
	peerBytes := string(p.MarshalBinary())

	conn := ps.rb.open()
	defer conn.Close()

	_ = conn.Send("MULTI")
	_ = conn.Send("HSET", sk, peerBytes, ct)
	_ = conn.Send("HSET", ihKey, sk, ct)
	reply, err := redis.Int64s(conn.Do("EXEC"))
	if err != nil {
		return err
	}

	if reply[0] == 1 { // The swarm or the peer was new.
		if _, err := conn.Do("INCR", peerCountKey(true, addr)); err != nil {
			return err
		}
	}

	if reply[1] == 1 { // The infohash or the swarm was new.
		if _, err := conn.Do("INCR", infohashCountKey(addr)); err != nil {
			return err
		}
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

	addr := p.AddrPort.Addr()
	peerBytes := string(p.MarshalBinary())
	delNum, err := redis.Int64(conn.Do("HDEL", swarmKey(ih, true, addr), peerBytes))
	if err != nil {
		return err
	}
	if delNum == 0 {
		return storage.ErrResourceDoesNotExist
	}

	if _, err := conn.Do("DECR", peerCountKey(true, addr)); err != nil {
		return err
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

	addr := p.AddrPort.Addr()
	sk := swarmKey(ih, false, addr)
	ihKey := sk[0]
	ct := timecache.NowUnixNano()
	peerBytes := string(p.MarshalBinary())

	conn := ps.rb.open()
	defer conn.Close()

	_ = conn.Send("MULTI")
	_ = conn.Send("HSET", sk, peerBytes, ct)
	_ = conn.Send("HSET", ihKey, sk, ct)
	reply, err := redis.Int64s(conn.Do("EXEC"))
	if err != nil {
		return err
	}

	if reply[0] == 1 { // The swarm or the peer was new.
		if _, err := conn.Do("INCR", peerCountKey(false, addr)); err != nil {
			return err
		}
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

	addr := p.AddrPort.Addr()
	peerBytes := string(p.MarshalBinary())
	delNum, err := redis.Int64(conn.Do("HDEL", swarmKey(ih, false, addr), peerBytes))
	if err != nil {
		return err
	}
	if delNum == 0 {
		return storage.ErrResourceDoesNotExist
	}

	if _, err := conn.Do("DECR", peerCountKey(false, addr)); err != nil {
		return err
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

	peerBytes := string(p.MarshalBinary())
	addr := p.AddrPort.Addr()
	leecherSK := swarmKey(ih, false, addr)
	seederSK := swarmKey(ih, true, addr)
	ihKey := seederSK[0]
	ct := timecache.NowUnixNano()

	conn := ps.rb.open()
	defer conn.Close()

	_ = conn.Send("MULTI")
	_ = conn.Send("HDEL", leecherSK, peerBytes)
	_ = conn.Send("HSET", seederSK, peerBytes, ct)
	_ = conn.Send("HSET", ihKey, seederSK, ct)
	reply, err := redis.Int64s(conn.Do("EXEC"))
	if err != nil {
		return err
	}
	if reply[0] == 1 {
		_, err = conn.Do("DECR", peerCountKey(false, addr))
		if err != nil {
			return err
		}
	}
	if reply[1] == 1 {
		_, err = conn.Do("INCR", peerCountKey(true, addr))
		if err != nil {
			return err
		}
	}
	if reply[2] == 1 {
		_, err = conn.Do("INCR", infohashCountKey(addr))
		if err != nil {
			return err
		}
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

	addr := announcer.AddrPort.Addr()
	leecherSK := swarmKey(ih, false, addr)
	seederSK := swarmKey(ih, true, addr)

	conn := ps.rb.open()
	defer conn.Close()

	leechers, err := conn.Do("HKEYS", leecherSK)
	if err != nil {
		return nil, err
	}
	conLeechers := leechers.([]interface{})

	seeders, err := conn.Do("HKEYS", seederSK)
	if err != nil {
		return nil, err
	}
	conSeeders := seeders.([]interface{})

	if len(conLeechers) == 0 && len(conSeeders) == 0 {
		return nil, storage.ErrResourceDoesNotExist
	}

	if seeder {
		// Append leechers as possible.
		for _, sk := range conLeechers {
			if numWant == 0 {
				break
			}

			peers = append(peers, bittorrent.PeerFromBytes(sk.([]byte)))
			numWant--
		}
	} else {
		// Append as many seeders as possible.
		for _, sk := range conSeeders {
			if numWant == 0 {
				break
			}

			peers = append(peers, bittorrent.PeerFromBytes(sk.([]byte)))
			numWant--
		}

		// Append leechers until we reach numWant.
		if numWant > 0 {
			announcerStr := string(announcer.MarshalBinary())
			for _, sk := range conLeechers {
				if sk == announcerStr {
					continue
				}

				if numWant == 0 {
					break
				}

				peers = append(peers, bittorrent.PeerFromBytes(sk.([]byte)))
				numWant--
			}
		}
	}

	return
}

func (ps *peerStore) ScrapeSwarm(ih bittorrent.InfoHash, p bittorrent.Peer) (resp bittorrent.Scrape) {
	select {
	case <-ps.closed:
		panic("attempted to interact with stopped redis store")
	default:
	}

	resp.InfoHash = ih
	leecherSK := swarmKey(ih, false, p.AddrPort.Addr())
	seederSK := swarmKey(ih, true, p.AddrPort.Addr())

	conn := ps.rb.open()
	defer conn.Close()

	leechersLen, err := redis.Int64(conn.Do("HLEN", leecherSK))
	if err != nil {
		log.Error("storage: Redis HLEN failure", log.Fields{
			"Hkey":  leecherSK,
			"error": err,
		})
		return
	}

	seedersLen, err := redis.Int64(conn.Do("HLEN", seederSK))
	if err != nil {
		log.Error("storage: Redis HLEN failure", log.Fields{
			"Hkey":  seederSK,
			"error": err,
		})
		return
	}

	resp.Incomplete = uint32(leechersLen)
	resp.Complete = uint32(seedersLen)

	return
}

// collectGarbage deletes all Peers from the PeerStore which are older than the
// cutoff time.
//
// This function must be able to execute while other methods on this interface
// are being executed in parallel.
//
// - The Delete(Seeder|Leecher) and GraduateLeecher methods never delete an
//	 infohash key from an addressFamily hash. They also never decrement the
//	 infohash counter.
// - The Put(Seeder|Leecher) and GraduateLeecher methods only ever add infohash
//	 keys to addressFamily hashes and increment the infohash counter.
// - The only method that deletes from the addressFamily hashes is
//	 collectGarbage, which also decrements the counters. That means that,
//	 even if a Delete(Seeder|Leecher) call removes the last peer from a swarm,
//	 the infohash counter is not changed and the infohash is left in the
//	 addressFamily hash until it will be cleaned up by collectGarbage.
// - collectGarbage must run regularly.
// - A WATCH ... MULTI ... EXEC block fails, if between the WATCH and the 'EXEC'
// 	 any of the watched keys have changed. The location of the 'MULTI' doesn't
//	 matter.
//
// We have to analyze four cases to prove our algorithm works. I'll characterize
// them by a tuple (number of peers in a swarm before WATCH, number of peers in
// the swarm during the transaction).
//
// 1. (0,0), the easy case: The swarm is empty, we watch the key, we execute
//	  HLEN and find it empty. We remove it and decrement the counter. It stays
//	  empty the entire time, the transaction goes through.
// 2. (1,n > 0): The swarm is not empty, we watch the key, we find it non-empty,
//	  we unwatch the key. All good. No transaction is made, no transaction fails.
// 3. (0,1): We have to analyze this in two ways.
// - If the change happens before the HLEN call, we will see that the swarm is
//	 not empty and start no transaction.
// - If the change happens after the HLEN, we will attempt a transaction and it
//   will fail. This is okay, the swarm is not empty, we will try cleaning it up
//   next time collectGarbage runs.
// 4. (1,0): Again, two ways:
// - If the change happens before the HLEN, we will see an empty swarm. This
//   situation happens if a call to Delete(Seeder|Leecher) removed the last
//	 peer asynchronously. We will attempt a transaction, but the transaction
//	 will fail. This is okay, the infohash key will remain in the addressFamily
//   hash, we will attempt to clean it up the next time 'collectGarbage` runs.
// - If the change happens after the HLEN, we will not even attempt to make the
//	 transaction. The infohash key will remain in the addressFamil hash and
//	 we'll attempt to clean it up the next time collectGarbage runs.
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

	for _, af := range addressFamilies {
		// list all infohashes in the group
		infohashesList, err := redis.Strings(conn.Do("HKEYS", "I"+af))
		if err != nil {
			return err
		}

		for _, ihStr := range infohashesList {
			isSeeder := len(ihStr) > 5 && ihStr[5:6] == "S"

			// list all (peer, timeout) pairs for the ih
			ihList, err := redis.Strings(conn.Do("HGETALL", ihStr))
			if err != nil {
				return err
			}

			var peerStr string
			var removedPeerCount int64
			for index, ihField := range ihList {
				if index%2 == 1 { // value
					mtime, err := strconv.ParseInt(ihField, 10, 64)
					if err != nil {
						return err
					}
					if mtime <= cutoffUnix {
						log.Debug("storage: deleting peer", log.Fields{
							"Peer": peerStr,
						})
						ret, err := redis.Int64(conn.Do("HDEL", ihStr, peerStr))
						if err != nil {
							return err
						}

						removedPeerCount += ret
					}
				} else { // key
					peerStr = ihField
				}
			}
			// DECR seeder/leecher counter
			decrCounter := af + "L"
			if isSeeder {
				decrCounter = af + "S"
			}
			if removedPeerCount > 0 {
				if _, err := conn.Do("DECRBY", decrCounter, removedPeerCount); err != nil {
					return err
				}
			}

			// use WATCH to avoid race condition
			// https://redis.io/topics/transactions
			_, err = conn.Do("WATCH", ihStr)
			if err != nil {
				return err
			}
			ihLen, err := redis.Int64(conn.Do("HLEN", ihStr))
			if err != nil {
				return err
			}
			if ihLen == 0 {
				// Empty hashes are not shown among existing keys,
				// in other words, it's removed automatically after `HDEL` the last field.
				//_, err := conn.Do("DEL", ihStr)

				_ = conn.Send("MULTI")
				_ = conn.Send("HDEL", af, ihStr)
				if isSeeder {
					_ = conn.Send("DECR", "I"+af)
				}
				_, err = redis.Values(conn.Do("EXEC"))
				if err != nil && !errors.Is(err, redis.ErrNil) {
					log.Error("storage: Redis EXEC failure", log.Fields{
						"addressFamily": af,
						"infohash":      ihStr,
						"error":         err,
					})
				}
			} else {
				if _, err = conn.Do("UNWATCH"); err != nil && !errors.Is(err, redis.ErrNil) {
					log.Error("storage: Redis UNWATCH failure", log.Fields{"error": err})
				}
			}
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
		log.Info("storage: exiting. reminder that chihaya does not clear redis data when exiting.")
		c.Done()
	}()

	return c.Result()
}

func (ps *peerStore) LogFields() log.Fields {
	return ps.cfg.LogFields()
}
