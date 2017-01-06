package redis

import (
	"encoding/binary"
	"net"
	"time"

	redigo "github.com/garyburd/redigo/redis"

	"github.com/chihaya/chihaya/bittorrent"
	"github.com/chihaya/chihaya/storage"
)

const (
	ipv4 = "4"
	ipv6 = "6"
)

// Config holds the configuration of a redis peerstore.
// KeyPrefix specifies the prefix that could optionally precede keys
// Instance specifies the redis database number to connect to (default 0)
// MaxNumWant is the maximum number of peers to return to announce
type Config struct {
	KeyPrefix    string        `yaml:"key_prefix"`
	Instance     int           `yaml:"instance"`
	MaxNumWant   int           `yaml:"max_numwant"`
	MaxIdle      int           `yaml:"max_idle"`
	Host         string        `yaml:"host"`
	Port         string        `yaml:"port"`
	PeerLifetime time.Duration `yaml:"peer_liftetime"`
}

type peerStore struct {
	connPool         *redigo.Pool
	closed           chan struct{}
	maxNumWant       int
	peerLifetime     time.Duration
	gcValidity       int
	seederKeyPrefix  string
	leecherKeyPrefix string
}

func newPool(server string, maxIdle int) redigo.Pool {
	return redigo.Pool{
		MaxIdle:     maxIdle,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redigo.Conn, error) {
			c, err := redigo.Dial("tcp", server)
			if err != nil {
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redigo.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}

// New creates a new peerstore backed by redis.
func New(cfg Config) (storage.PeerStore, error) {
	pool := newPool(cfg.Host+":"+cfg.Port, cfg.MaxIdle)
	conn := pool.Get()
	defer conn.Close()

	if cfg.Instance != 0 {
		conn.Do("SELECT", cfg.Instance)
	}

	ps := &peerStore{
		connPool:         &pool,
		closed:           make(chan struct{}),
		maxNumWant:       cfg.MaxNumWant,
		peerLifetime:     cfg.PeerLifetime,
		seederKeyPrefix:  cfg.KeyPrefix + "seeder",
		leecherKeyPrefix: cfg.KeyPrefix + "leecher",
	}

	return ps, nil
}

type serializedPeer string

func newPeerKey(p bittorrent.Peer) serializedPeer {
	b := make([]byte, 20+2+len(p.IP))
	copy(b[:20], p.ID[:])
	binary.BigEndian.PutUint16(b[20:22], p.Port)
	copy(b[22:], p.IP)

	return serializedPeer(b)
}

func panicIfClosed(closed <-chan struct{}) {
	select {
	case <-closed:
		panic("attempted to interact with stopped redis store")
	default:
	}
}

func ipType(ip net.IP) string {
	if len(ip) == net.IPv6len {
		return ipv6
	}
	return ipv4
}

func (s *peerStore) PutSeeder(infoHash bittorrent.InfoHash, p bittorrent.Peer) error {
	panicIfClosed(s.closed)

	pk := newPeerKey(p)
	return addPeer(s, infoHash, s.seederKeyPrefix+ipType(p.IP), pk)
}

func (s *peerStore) DeleteSeeder(infoHash bittorrent.InfoHash, p bittorrent.Peer) error {
	panicIfClosed(s.closed)
	pk := newPeerKey(p)
	return removePeers(s, infoHash, s.seederKeyPrefix+ipType(p.IP), pk)
}

func (s *peerStore) PutLeecher(infoHash bittorrent.InfoHash, p bittorrent.Peer) error {
	panicIfClosed(s.closed)
	pk := newPeerKey(p)
	return addPeer(s, infoHash, s.leecherKeyPrefix+ipType(p.IP), pk)
}

func (s *peerStore) DeleteLeecher(infoHash bittorrent.InfoHash, p bittorrent.Peer) error {
	panicIfClosed(s.closed)
	pk := newPeerKey(p)
	return removePeers(s, infoHash, s.leecherKeyPrefix+ipType(p.IP), pk)
}

func (s *peerStore) GraduateLeecher(infoHash bittorrent.InfoHash, p bittorrent.Peer) error {
	panicIfClosed(s.closed)
	err := s.PutSeeder(infoHash, p)
	if err != nil {
		return err
	}
	err = s.DeleteLeecher(infoHash, p)
	if err != nil {
		return err
	}
	return nil
}

// Announce as many peers as possible based on the announcer being
// a seeder or leecher
func (s *peerStore) AnnouncePeers(infoHash bittorrent.InfoHash, seeder bool, numWant int, announcer bittorrent.Peer) (peers []bittorrent.Peer, err error) {
	panicIfClosed(s.closed)
	if numWant > s.maxNumWant {
		numWant = s.maxNumWant
	}

	if seeder {
		peers, err = getPeers(s, infoHash, s.leecherKeyPrefix+ipType(announcer.IP), numWant, peers, bittorrent.Peer{})
		if err != nil {
			return nil, err
		}
	} else {
		peers, err = getPeers(s, infoHash, s.seederKeyPrefix+ipType(announcer.IP), numWant, peers, bittorrent.Peer{})
		if err != nil {
			return nil, err
		}
		if len(peers) < numWant {
			peers, err = getPeers(s, infoHash, s.leecherKeyPrefix+ipType(announcer.IP), numWant, peers, announcer)
		}
	}
	return peers, nil
}

func (s *peerStore) ScrapeSwarm(infoHash bittorrent.InfoHash, v6 bool) (resp bittorrent.Scrape) {
	panicIfClosed(s.closed)

	ipType := ipv4
	if v6 {
		ipType = ipv6
	}
	complete, err := getPeersLength(s, infoHash, s.seederKeyPrefix+ipType)
	if err != nil {
		return
	}
	resp.Complete = uint32(complete)
	incomplete, err := getPeersLength(s, infoHash, s.leecherKeyPrefix+ipType)
	if err != nil {
		return
	}
	resp.Incomplete = uint32(incomplete)
	return
}

func (s *peerStore) Stop() <-chan error {
	toReturn := make(chan error)
	go func() {
		close(s.closed)
		close(toReturn)
	}()
	return toReturn
}
