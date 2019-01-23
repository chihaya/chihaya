// Package redis implements the storage interface for a Chihaya
// BitTorrent tracker keeping peer data in redis with hash.
// There two categories of hash:
//
// - IPv{4,6}_{L,S}_infohash
//	To save peers that hold the infohash, used for fast searching,
//  deleting, and timeout handling
//
// - IPv{4,6}
//  To save all the infohashes, used for garbage collection,
//	metrics aggregation and leecher graduation
//
// Tree keys are used to record the count of swarms, seeders
// and leechers for each group (IPv4, IPv6).
//
// - IPv{4,6}_infohash_count
//	To record the number of infohashes.
//
// - IPv{4,6}_S_count
//	To record the number of seeders.
//
// - IPv{4,6}_L_count
//	To record the number of leechers.
package redis

import (
	"encoding/binary"
	"net"
	"strconv"
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

type peerStore struct {
	cfg Config
	rb  *redisBackend

	closed chan struct{}
	wg     sync.WaitGroup
}

func (ps *peerStore) groups() []string {
	return []string{bittorrent.IPv4.String(), bittorrent.IPv6.String()}
}

func (ps *peerStore) leecherInfohashKey(af, ih string) string {
	return af + "_L_" + ih
}

func (ps *peerStore) seederInfohashKey(af, ih string) string {
	return af + "_S_" + ih
}

func (ps *peerStore) infohashCountKey(af string) string {
	return af + "_infohash_count"
}

func (ps *peerStore) seederCountKey(af string) string {
	return af + "_S_count"
}

func (ps *peerStore) leecherCountKey(af string) string {
	return af + "_L_count"
}

// populateProm aggregates metrics over all groups and then posts them to
// prometheus.
func (ps *peerStore) populateProm() {
	var numInfohashes, numSeeders, numLeechers int64

	conn := ps.rb.open()
	defer conn.Close()

	for _, group := range ps.groups() {
		if n, err := redis.Int64(conn.Do("GET", ps.infohashCountKey(group))); err != nil && err != redis.ErrNil {
			log.Error("storage: GET counter failure", log.Fields{
				"key":   ps.infohashCountKey(group),
				"error": err,
			})
		} else {
			numInfohashes += n
		}
		if n, err := redis.Int64(conn.Do("GET", ps.seederCountKey(group))); err != nil && err != redis.ErrNil {
			log.Error("storage: GET counter failure", log.Fields{
				"key":   ps.seederCountKey(group),
				"error": err,
			})
		} else {
			numSeeders += n
		}
		if n, err := redis.Int64(conn.Do("GET", ps.leecherCountKey(group))); err != nil && err != redis.ErrNil {
			log.Error("storage: GET counter failure", log.Fields{
				"key":   ps.leecherCountKey(group),
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
	addressFamily := p.IP.AddressFamily.String()
	log.Debug("storage: PutSeeder", log.Fields{
		"InfoHash": ih.String(),
		"Peer":     p,
	})

	select {
	case <-ps.closed:
		panic("attempted to interact with stopped redis store")
	default:
	}

	pk := newPeerKey(p)

	encodedSeederInfoHash := ps.seederInfohashKey(addressFamily, ih.String())
	ct := ps.getClock()

	conn := ps.rb.open()
	defer conn.Close()

	conn.Send("MULTI")
	conn.Send("HSET", encodedSeederInfoHash, pk, ct)
	conn.Send("HSET", addressFamily, encodedSeederInfoHash, ct)
	reply, err := redis.Int64s(conn.Do("EXEC"))
	if err != nil {
		return err
	}

	// pk is a new field.
	if reply[0] == 1 {
		_, err = conn.Do("INCR", ps.seederCountKey(addressFamily))
		if err != nil {
			return err
		}
	}
	// encodedSeederInfoHash is a new field.
	if reply[1] == 1 {
		_, err = conn.Do("INCR", ps.infohashCountKey(addressFamily))
		if err != nil {
			return err
		}
	}

	return nil
}

func (ps *peerStore) DeleteSeeder(ih bittorrent.InfoHash, p bittorrent.Peer) error {
	addressFamily := p.IP.AddressFamily.String()
	log.Debug("storage: DeleteSeeder", log.Fields{
		"InfoHash": ih.String(),
		"Peer":     p,
	})

	select {
	case <-ps.closed:
		panic("attempted to interact with stopped redis store")
	default:
	}

	pk := newPeerKey(p)

	conn := ps.rb.open()
	defer conn.Close()

	encodedSeederInfoHash := ps.seederInfohashKey(addressFamily, ih.String())

	delNum, err := redis.Int64(conn.Do("HDEL", encodedSeederInfoHash, pk))
	if err != nil {
		return err
	}
	if delNum == 0 {
		return storage.ErrResourceDoesNotExist
	}
	if _, err := conn.Do("DECR", ps.seederCountKey(addressFamily)); err != nil {
		return err
	}

	return nil
}

func (ps *peerStore) PutLeecher(ih bittorrent.InfoHash, p bittorrent.Peer) error {
	addressFamily := p.IP.AddressFamily.String()
	log.Debug("storage: PutLeecher", log.Fields{
		"InfoHash": ih.String(),
		"Peer":     p,
	})

	select {
	case <-ps.closed:
		panic("attempted to interact with stopped redis store")
	default:
	}

	// Update the peer in the swarm.
	encodedLeecherInfoHash := ps.leecherInfohashKey(addressFamily, ih.String())
	pk := newPeerKey(p)
	ct := ps.getClock()

	conn := ps.rb.open()
	defer conn.Close()

	conn.Send("MULTI")
	conn.Send("HSET", encodedLeecherInfoHash, pk, ct)
	conn.Send("HSET", addressFamily, encodedLeecherInfoHash, ct)
	reply, err := redis.Int64s(conn.Do("EXEC"))
	if err != nil {
		return err
	}
	// pk is a new field.
	if reply[0] == 1 {
		_, err = conn.Do("INCR", ps.leecherCountKey(addressFamily))
		if err != nil {
			return err
		}
	}
	return nil
}

func (ps *peerStore) DeleteLeecher(ih bittorrent.InfoHash, p bittorrent.Peer) error {
	addressFamily := p.IP.AddressFamily.String()
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

	pk := newPeerKey(p)
	encodedLeecherInfoHash := ps.leecherInfohashKey(addressFamily, ih.String())

	delNum, err := redis.Int64(conn.Do("HDEL", encodedLeecherInfoHash, pk))
	if err != nil {
		return err
	}
	if delNum == 0 {
		return storage.ErrResourceDoesNotExist
	}
	if _, err := conn.Do("DECR", ps.leecherCountKey(addressFamily)); err != nil {
		return err
	}

	return nil
}

func (ps *peerStore) GraduateLeecher(ih bittorrent.InfoHash, p bittorrent.Peer) error {
	addressFamily := p.IP.AddressFamily.String()
	log.Debug("storage: GraduateLeecher", log.Fields{
		"InfoHash": ih.String(),
		"Peer":     p,
	})

	select {
	case <-ps.closed:
		panic("attempted to interact with stopped redis store")
	default:
	}

	encodedInfoHash := ih.String()
	encodedLeecherInfoHash := ps.leecherInfohashKey(addressFamily, encodedInfoHash)
	encodedSeederInfoHash := ps.seederInfohashKey(addressFamily, encodedInfoHash)
	pk := newPeerKey(p)
	ct := ps.getClock()

	conn := ps.rb.open()
	defer conn.Close()

	conn.Send("MULTI")
	conn.Send("HDEL", encodedLeecherInfoHash, pk)
	conn.Send("HSET", encodedSeederInfoHash, pk, ct)
	conn.Send("HSET", addressFamily, encodedSeederInfoHash, ct)
	reply, err := redis.Int64s(conn.Do("EXEC"))
	if err != nil {
		return err
	}
	if reply[0] == 1 {
		_, err = conn.Do("DECR", ps.leecherCountKey(addressFamily))
		if err != nil {
			return err
		}
	}
	if reply[1] == 1 {
		_, err = conn.Do("INCR", ps.seederCountKey(addressFamily))
		if err != nil {
			return err
		}
	}
	if reply[2] == 1 {
		_, err = conn.Do("INCR", ps.infohashCountKey(addressFamily))
		if err != nil {
			return err
		}
	}

	return nil
}

func (ps *peerStore) AnnouncePeers(ih bittorrent.InfoHash, seeder bool, numWant int, announcer bittorrent.Peer) (peers []bittorrent.Peer, err error) {
	addressFamily := announcer.IP.AddressFamily.String()
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

	encodedInfoHash := ih.String()
	encodedLeecherInfoHash := ps.leecherInfohashKey(addressFamily, encodedInfoHash)
	encodedSeederInfoHash := ps.seederInfohashKey(addressFamily, encodedInfoHash)

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

	return
}

func (ps *peerStore) ScrapeSwarm(ih bittorrent.InfoHash, af bittorrent.AddressFamily) (resp bittorrent.Scrape) {
	select {
	case <-ps.closed:
		panic("attempted to interact with stopped redis store")
	default:
	}

	resp.InfoHash = ih
	addressFamily := af.String()
	encodedInfoHash := ih.String()
	encodedLeecherInfoHash := ps.leecherInfohashKey(addressFamily, encodedInfoHash)
	encodedSeederInfoHash := ps.seederInfohashKey(addressFamily, encodedInfoHash)

	conn := ps.rb.open()
	defer conn.Close()

	leechersLen, err := redis.Int64(conn.Do("HLEN", encodedLeecherInfoHash))
	if err != nil {
		log.Error("storage: Redis HLEN failure", log.Fields{
			"Hkey":  encodedLeecherInfoHash,
			"error": err,
		})
		return
	}

	seedersLen, err := redis.Int64(conn.Do("HLEN", encodedSeederInfoHash))
	if err != nil {
		log.Error("storage: Redis HLEN failure", log.Fields{
			"Hkey":  encodedSeederInfoHash,
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

	for _, group := range ps.groups() {
		// list all infohashes in the group
		infohashesList, err := redis.Strings(conn.Do("HKEYS", group))
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

			var pk serializedPeer
			var removedPeerCount int64
			for index, ihField := range ihList {
				if index%2 == 1 { // value
					mtime, err := strconv.ParseInt(ihField, 10, 64)
					if err != nil {
						return err
					}
					if mtime <= cutoffUnix {
						log.Debug("storage: deleting peer", log.Fields{
							"Peer": decodePeerKey(pk).String(),
						})
						ret, err := redis.Int64(conn.Do("HDEL", ihStr, pk))
						if err != nil {
							return err
						}

						removedPeerCount += ret
					}
				} else { // key
					pk = serializedPeer([]byte(ihField))
				}
			}
			// DECR seeder/leecher counter
			decrCounter := ps.leecherCountKey(group)
			if isSeeder {
				decrCounter = ps.seederCountKey(group)
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

				conn.Send("MULTI")
				conn.Send("HDEL", group, ihStr)
				if isSeeder {
					conn.Send("DECR", ps.infohashCountKey(group))
				}
				_, err = redis.Values(conn.Do("EXEC"))
				if err != nil && err != redis.ErrNil {
					log.Error("storage: Redis EXEC failure", log.Fields{
						"group":    group,
						"infohash": ihStr,
						"error":    err,
					})
				}
			} else {
				if _, err = conn.Do("UNWATCH"); err != nil && err != redis.ErrNil {
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
		log.Info("storage: exiting. chihaya does not clear data in redis when exiting. chihaya keys have prefix 'IPv{4,6}_'.")
		c.Done()
	}()

	return c.Result()
}

func (ps *peerStore) LogFields() log.Fields {
	return ps.cfg.LogFields()
}
