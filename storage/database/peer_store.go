// Package database implements the storage interface for a Chihaya
// BitTorrent tracker keeping peer data in memory.
package database

// Workaround, sqlite is being noisy.
// #cgo CFLAGS: -Wno-return-local-addr

import (
	"encoding/binary"
	"encoding/hex"
	"net"
	"sync"
	"time"

	yaml "gopkg.in/yaml.v2"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/chihaya/chihaya/bittorrent"
	"github.com/chihaya/chihaya/pkg/log"
	"github.com/chihaya/chihaya/pkg/stop"
	"github.com/chihaya/chihaya/storage"
)

// Name is the name by which this peer store is registered with Chihaya.
const Name = "database"

// Default config constants.
const (
	defaultPrometheusReportingInterval = time.Second * 1
	defaultGarbageCollectionInterval   = time.Minute * 3
	defaultPeerLifetime                = time.Minute * 30
	defaultDsn                         = "data/chihaya.sqlite"
)

func init() {
	// Register the storage drivers.
	storage.RegisterDriver("postgres", postgresDriver{})
	storage.RegisterDriver("sqlite", sqliteDriver{})
}

type postgresDriver struct{}
type sqliteDriver struct{}

func (d postgresDriver) NewPeerStore(icfg interface{}) (storage.PeerStore, error) {
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

	return NewPostgres(cfg)
}

func (d sqliteDriver) NewPeerStore(icfg interface{}) (storage.PeerStore, error) {
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

	return NewSqlite(cfg)
}

// Config holds the configuration of a memory PeerStore.
type Config struct {
	GarbageCollectionInterval   time.Duration `yaml:"gc_interval"`
	PrometheusReportingInterval time.Duration `yaml:"prometheus_reporting_interval"`
	PeerLifetime                time.Duration `yaml:"peer_lifetime"`
	Dsn                         string        `yaml:"dsn"`
}

// LogFields renders the current config as a set of Logrus fields.
func (cfg Config) LogFields() log.Fields {
	return log.Fields{
		"name":               Name,
		"gcInterval":         cfg.GarbageCollectionInterval,
		"promReportInterval": cfg.PrometheusReportingInterval,
		"peerLifetime":       cfg.PeerLifetime,
		"dsn":                cfg.Dsn,
	}
}

// Validate sanity checks values set in a config and returns a new config with
// default values replacing anything that is invalid.
//
// This function warns to the logger when a value is changed.
func (cfg Config) Validate() Config {
	validcfg := cfg

	if cfg.Dsn == "" {
		validcfg.Dsn = defaultDsn
		log.Warn("falling back to default dsn", log.Fields{
			"name":     Name + ".dsn",
			"provided": cfg.Dsn,
			"default":  validcfg.Dsn,
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

// New creates a new PeerStore backed by a postgres database.
func NewPostgres(provided Config) (storage.PeerStore, error) {
	cfg := provided.Validate()

	db, err := gorm.Open(postgres.Open(provided.Dsn), nil)
	if err != nil {
		log.Fatal("Unable to connect to Postgres database:", log.Fields{"reason": err})
	}

	ps := &peerStore{
		cfg:    cfg,
		db:     db,
		closed: make(chan struct{}),
	}

	err = db.AutoMigrate(&ipv4Seeder{}, &ipv4Leecher{}, &ipv6Seeder{}, &ipv6Leecher{})
	if err != nil {
		log.Fatal("Unable to migrate database:", log.Fields{"reason": err})
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
				_ = ps.collectGarbage(before)
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

// New creates a new PeerStore backed by an sqlite database.
func NewSqlite(provided Config) (storage.PeerStore, error) {
	cfg := provided.Validate()

	db, err := gorm.Open(sqlite.Open(provided.Dsn), nil)

	if err != nil {
		log.Fatal("Unable to open the Sqlite database:", log.Fields{"reason": err})
	}

	ps := &peerStore{
		cfg:    cfg,
		db:     db,
		closed: make(chan struct{}),
	}

	err = db.AutoMigrate(&ipv4Seeder{}, &ipv4Leecher{}, &ipv6Seeder{}, &ipv6Leecher{})
	if err != nil {
		log.Fatal("Unable to migrate database:", log.Fields{"reason": err})
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
				_ = ps.collectGarbage(before)
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

func newPeerKey(p bittorrent.Peer) string {
	b := make([]byte, 20+2+len(p.IP.IP))
	copy(b[:20], p.ID[:])
	binary.BigEndian.PutUint16(b[20:22], p.Port)
	copy(b[22:], p.IP.IP)

	return hex.EncodeToString(b)
}

func decodePeerKey(pk string) bittorrent.Peer {
	bytes, err := hex.DecodeString(pk)
	if err != nil {
		panic("non-hex string in decodePeerKey")
	}

	peer := bittorrent.Peer{
		ID:   bittorrent.PeerIDFromString(string(bytes[:20])),
		Port: binary.BigEndian.Uint16([]byte(bytes[20:22])),
		IP:   bittorrent.IP{IP: net.IP(bytes[22:])},
	}

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
	cfg    Config
	db     *gorm.DB
	closed chan struct{}
	wg     sync.WaitGroup
}

var _ storage.PeerStore = &peerStore{}

// populateProm aggregates metrics over all shards and then posts them to
// prometheus.
func (ps *peerStore) populateProm() {
	ipv4Seeders := 0
	ipv4Leechers := 0
	ipv6Seeders := 0
	ipv6Leechers := 0

	allInfohashes := 0

	ps.db.Select("COUNT(*)").Table("ipv4_seeders").Row().Scan(&ipv4Seeders)
	ps.db.Select("COUNT(*)").Table("ipv4_leechers").Row().Scan(&ipv4Leechers)
	ps.db.Select("COUNT(*)").Table("ipv6_seeders").Row().Scan(&ipv6Seeders)
	ps.db.Select("COUNT(*)").Table("ipv6_leechers").Row().Scan(&ipv6Leechers)

	ps.db.Select("COUNT(*)").Table("ipv4_seeders").Row().Scan(&ipv4Seeders)
	ps.db.Select("COUNT(*)").Table("ipv4_leechers").Row().Scan(&ipv4Leechers)
	ps.db.Select("COUNT(*)").Table("ipv6_seeders").Row().Scan(&ipv6Seeders)
	ps.db.Select("COUNT(*)").Table("ipv6_leechers").Row().Scan(&ipv6Leechers)

	ps.db.Select("COUNT(DISTINCT(*))").Table(`(
		SELECT DISTINCT(info_hash) FROM ipv4_seeders
		UNION
		SELECT DISTINCT(info_hash) FROM ipv4_leechers
		UNION
		SELECT DISTINCT(info_hash) FROM ipv6_seeders
		UNION
		SELECT DISTINCT(info_hash) FROM ipv6_leechers
	)`).Row().Scan(&allInfohashes)

	storage.PromInfohashesCount.Set(float64(allInfohashes))
	storage.PromSeedersCount.Set(float64(ipv4Seeders + ipv6Seeders))
	storage.PromLeechersCount.Set(float64(ipv4Leechers + ipv6Leechers))
}

// recordGCDuration records the duration of a GC sweep.
func recordGCDuration(duration time.Duration) {
	storage.PromGCDurationMilliseconds.Observe(float64(duration.Nanoseconds()) / float64(time.Millisecond))
}

type peer struct {
	PeerKey   string `gorm:"primary_key"`
	InfoHash  string `gorm:"index"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ipv4Seeder peer
type ipv4Leecher peer
type ipv6Seeder peer
type ipv6Leecher peer

func (ps *peerStore) PutSeeder(ih bittorrent.InfoHash, p bittorrent.Peer) error {
	select {
	case <-ps.closed:
		panic("attempted to interact with stopped datatabase store")
	default:
	}

	peerKey := newPeerKey(p)

	switch p.IP.AddressFamily {
	case bittorrent.IPv4:
		row := &ipv4Seeder{PeerKey: peerKey, InfoHash: ih.String()}
		if err := ps.db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&row).Error; err != nil {
			return err
		}
	case bittorrent.IPv6:
		row := &ipv6Seeder{PeerKey: peerKey, InfoHash: ih.String()}
		if err := ps.db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&row).Error; err != nil {
			return err
		}
	}

	return nil
}

func (ps *peerStore) DeleteSeeder(ih bittorrent.InfoHash, p bittorrent.Peer) error {
	select {
	case <-ps.closed:
		panic("attempted to interact with stopped datatabase store")
	default:
	}

	peerKey := newPeerKey(p)

	switch p.IP.AddressFamily {
	case bittorrent.IPv4:
		row := &ipv4Seeder{PeerKey: peerKey, InfoHash: ih.String()}
		tx := ps.db.Delete(&row)
		if tx.Error != nil {
			return tx.Error
		}
		if tx.RowsAffected == 0 {
			return storage.ErrResourceDoesNotExist
		}
	case bittorrent.IPv6:
		row := &ipv6Seeder{PeerKey: peerKey, InfoHash: ih.String()}
		tx := ps.db.Delete(&row)
		if tx.Error != nil {
			return tx.Error
		}
		if tx.RowsAffected == 0 {
			return storage.ErrResourceDoesNotExist
		}
	}

	return nil
}

func (ps *peerStore) PutLeecher(ih bittorrent.InfoHash, p bittorrent.Peer) error {
	select {
	case <-ps.closed:
		panic("attempted to interact with stopped datatabase store")
	default:
	}

	peerKey := newPeerKey(p)

	switch p.IP.AddressFamily {
	case bittorrent.IPv4:
		row := &ipv4Leecher{PeerKey: peerKey, InfoHash: ih.String()}
		if err := ps.db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&row).Error; err != nil {
			return err
		}
	case bittorrent.IPv6:
		row := &ipv6Leecher{PeerKey: peerKey, InfoHash: ih.String()}
		if err := ps.db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&row).Error; err != nil {
			return err
		}
	}

	return nil
}

func (ps *peerStore) DeleteLeecher(ih bittorrent.InfoHash, p bittorrent.Peer) error {
	select {
	case <-ps.closed:
		panic("attempted to interact with stopped datatabase store")
	default:
	}

	peerKey := newPeerKey(p)

	switch p.IP.AddressFamily {
	case bittorrent.IPv4:
		row := &ipv4Leecher{PeerKey: peerKey, InfoHash: ih.String()}
		tx := ps.db.Delete(&row)
		if tx.Error != nil {
			return tx.Error
		}
		if tx.RowsAffected == 0 {
			return storage.ErrResourceDoesNotExist
		}
	case bittorrent.IPv6:
		row := &ipv6Leecher{PeerKey: peerKey, InfoHash: ih.String()}
		tx := ps.db.Delete(&row)
		if tx.Error != nil {
			return tx.Error
		}
		if tx.RowsAffected == 0 {
			return storage.ErrResourceDoesNotExist
		}
	}

	return nil
}

func (ps *peerStore) GraduateLeecher(ih bittorrent.InfoHash, p bittorrent.Peer) error {
	select {
	case <-ps.closed:
		panic("attempted to interact with stopped datatabase store")
	default:
	}

	err := ps.PutSeeder(ih, p)
	if err != nil {
		return err
	}

	err = ps.DeleteLeecher(ih, p)
	if err != nil {
		return err
	}

	return nil
}

func (ps *peerStore) AnnouncePeers(ih bittorrent.InfoHash, seeder bool, numWant int, announcer bittorrent.Peer) (peers []bittorrent.Peer, err error) {
	select {
	case <-ps.closed:
		panic("attempted to interact with stopped datatabase store")
	default:
	}

	originalNumWant := numWant

	if seeder {
		// Append as many leechers as possible.
		switch announcer.IP.AddressFamily {
		case bittorrent.IPv4:
			ipv4Leechers := []ipv4Leecher{}
			if err := ps.db.Select("peer_key").Limit(numWant).Find(&ipv4Leechers, "info_hash = ?", ih.String()).Error; err != nil {
				return peers, err
			}

			for _, leecher := range ipv4Leechers {
				peers = append(peers, decodePeerKey(leecher.PeerKey))
				numWant--
			}
		case bittorrent.IPv6:
			ipv6Leechers := []ipv6Leecher{}
			if err := ps.db.Select("peer_key").Limit(numWant).Find(&ipv6Leechers, "info_hash = ?", ih.String()).Error; err != nil {
				return peers, err
			}

			for _, leecher := range ipv6Leechers {
				peers = append(peers, decodePeerKey(leecher.PeerKey))
				numWant--
			}
		}
	} else {
		// Append as many seeders as possible.
		switch announcer.IP.AddressFamily {
		case bittorrent.IPv4:
			ipv4Seeders := []ipv4Seeder{}
			if err := ps.db.Select("peer_key").Limit(numWant).Find(&ipv4Seeders, "info_hash = ?", ih.String()).Error; err != nil {
				return peers, err
			}

			for _, seeder := range ipv4Seeders {
				peers = append(peers, decodePeerKey(seeder.PeerKey))
				numWant--
			}
		case bittorrent.IPv6:
			ipv6Seeders := []ipv6Seeder{}
			if err := ps.db.Select("peer_key").Limit(numWant).Find(&ipv6Seeders, "info_hash = ?", ih.String()).Error; err != nil {
				return peers, err
			}

			for _, seeder := range ipv6Seeders {
				peers = append(peers, decodePeerKey(seeder.PeerKey))
				numWant--
			}
		}
	}

	// Fill the rest with leechers to exchaut numWant.
	seenMyself := false

	if numWant > 0 {
		pk := newPeerKey(announcer)

		switch announcer.IP.AddressFamily {
		case bittorrent.IPv4:
			ipv4Leechers := []ipv4Leecher{}
			if err := ps.db.Select("peer_key").Limit(numWant).Find(&ipv4Leechers, "info_hash = ?", ih.String()).Error; err != nil {
				return peers, err
			}

			for _, leecher := range ipv4Leechers {
				if leecher.PeerKey == pk {
					seenMyself = true
					continue
				}

				peers = append(peers, decodePeerKey(leecher.PeerKey))
				numWant--
			}
		case bittorrent.IPv6:
			ipv6Leechers := []ipv6Leecher{}
			if err := ps.db.Select("peer_key").Limit(numWant).Find(&ipv6Leechers, "info_hash = ?", ih.String()).Error; err != nil {
				return peers, err
			}

			for _, leecher := range ipv6Leechers {
				if leecher.PeerKey == pk {
					seenMyself = true
					continue
				}

				peers = append(peers, decodePeerKey(leecher.PeerKey))
				numWant--
			}
		}
	}

	if numWant == originalNumWant && !seenMyself {
		return nil, storage.ErrResourceDoesNotExist
	}

	return
}

func (ps *peerStore) ScrapeSwarm(ih bittorrent.InfoHash, addressFamily bittorrent.AddressFamily) (resp bittorrent.Scrape) {
	select {
	case <-ps.closed:
		panic("attempted to interact with stopped datatabase store")
	default:
	}

	resp.InfoHash = ih

	switch addressFamily {
	case bittorrent.IPv4:
		ps.db.Select("COUNT(*)").Table("ipv4_seeders").Where("info_hash = ?", ih.String()).Row().Scan(&resp.Complete)
		ps.db.Select("COUNT(*)").Table("ipv4_leechers").Where("info_hash = ?", ih.String()).Row().Scan(&resp.Incomplete)
	case bittorrent.IPv6:
		ps.db.Select("COUNT(*)").Table("ipv6_seeders").Where("info_hash = ?", ih.String()).Row().Scan(&resp.Complete)
		ps.db.Select("COUNT(*)").Table("ipv6_leechers").Where("info_hash = ?", ih.String()).Row().Scan(&resp.Incomplete)
	}

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

	start := time.Now()

	if err := ps.db.Delete(ipv4Seeder{}, "updated_at < ?", cutoff).Error; err != nil {
		return err
	}

	if err := ps.db.Delete(ipv4Leecher{}, "updated_at < ?", cutoff).Error; err != nil {
		return err
	}

	if err := ps.db.Delete(ipv6Seeder{}, "updated_at < ?", cutoff).Error; err != nil {
		return err
	}

	if err := ps.db.Delete(ipv6Leecher{}, "updated_at < ?", cutoff).Error; err != nil {
		return err
	}

	recordGCDuration(time.Since(start))

	return nil
}

func (ps *peerStore) Stop() stop.Result {
	c := make(stop.Channel)

	go func() {
		close(ps.closed)
		ps.wg.Wait()
		c.Done()
	}()

	return c.Result()
}

func (ps *peerStore) LogFields() log.Fields {
	return ps.cfg.LogFields()
}
