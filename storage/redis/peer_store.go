package redis

import (
	"encoding/binary"
	"log"
	"time"

	"github.com/chihaya/chihaya/bittorrent"
	"github.com/chihaya/chihaya/storage"
	redigo "github.com/garyburd/redigo/redis"
)

// Config holds the configuration of a redis peerstore.
// KeyPrefix specifies the prefix that could optionally precede keys
// Instance specifies the redis database number to connect to(default 0)
// max_numwant is the maximum number of peers to return to announce
type Config struct {
	KeyPrefix    string        `yaml:"key_prefix"`
	Instance     int           `yaml:"instance"`
	MaxNumWant   int           `yaml:"max_numwant"`
	Host         string        `yaml:"host"`
	Port         string        `yaml:"port"`
	PeerLifetime time.Duration `yaml:"peer_liftetime"`
}

type peerStore struct {
	conn             redigo.Conn
	closed           chan struct{}
	maxNumWant       int
	peerLifetime     time.Duration
	gcValidity       int
	seederKeyPrefix  string
	leecherKeyPrefix string
}

//New creates a new peerstore backed by redis
func New(cfg Config) (storage.PeerStore, error) {
	conn, err := redigo.Dial("tcp", cfg.Host+":"+cfg.Port)
	if err != nil {
		log.Fatal("Connection failed:" + err.Error())
		return nil, err
	}

	if cfg.Instance != 0 {
		conn.Do("SELECT", cfg.Instance)
	}

	ps := &peerStore{
		conn:             conn,
		closed:           make(chan struct{}),
		maxNumWant:       cfg.MaxNumWant,
		peerLifetime:     cfg.PeerLifetime,
		seederKeyPrefix:  addKeyPrefix(cfg.KeyPrefix, "seeder"),
		leecherKeyPrefix: addKeyPrefix(cfg.KeyPrefix, "leecher"),
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

func addKeyPrefix(namespacePrefix string, command string) string {
	if namespacePrefix != "" {
		return namespacePrefix + ":" + command
	}
	return command
}

func (s *peerStore) PutSeeder(infoHash bittorrent.InfoHash,
	p bittorrent.Peer) error {
	panicIfClosed(s.closed)

	pk := newPeerKey(p)
	return addPeer(s, infoHash, s.seederKeyPrefix, pk)
}

func (s *peerStore) DeleteSeeder(infoHash bittorrent.InfoHash,
	p bittorrent.Peer) error {
	panicIfClosed(s.closed)
	pk := newPeerKey(p)
	return removePeers(s, infoHash, s.seederKeyPrefix, pk)
}

func (s *peerStore) PutLeecher(infoHash bittorrent.InfoHash,
	p bittorrent.Peer) error {
	panicIfClosed(s.closed)
	pk := newPeerKey(p)
	return addPeer(s, infoHash, s.leecherKeyPrefix, pk)

}

func (s *peerStore) DeleteLeecher(infoHash bittorrent.InfoHash,
	p bittorrent.Peer) error {
	panicIfClosed(s.closed)
	pk := newPeerKey(p)
	return removePeers(s, infoHash, s.leecherKeyPrefix, pk)

}

func (s *peerStore) GraduateLeecher(infoHash bittorrent.InfoHash,
	p bittorrent.Peer) error {
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
func (s *peerStore) AnnouncePeers(infoHash bittorrent.InfoHash, seeder bool,
	numWant int, announcer bittorrent.Peer) (peers []bittorrent.Peer,
	err error) {
	panicIfClosed(s.closed)
	if numWant > s.maxNumWant {
		numWant = s.maxNumWant
	}

	peers = []bittorrent.Peer{}
	if seeder {
		peers, err = getPeers(s, infoHash, s.leecherKeyPrefix,
			numWant, peers, bittorrent.Peer{})
		if err != nil {
			return nil, err
		}
	} else {
		peers, err = getPeers(s, infoHash, s.seederKeyPrefix,
			numWant, peers, bittorrent.Peer{})
		if err != nil {
			return nil, err
		}
		if len(peers) < numWant {
			peers, err = getPeers(s, infoHash, s.leecherKeyPrefix,
				numWant, peers, announcer)
		}
	}
	return peers, nil
}

func (s *peerStore) ScrapeSwarm(infoHash bittorrent.InfoHash, v6 bool) (
	resp bittorrent.Scrape) {
	panicIfClosed(s.closed)
	complete, err := getPeersLength(s, infoHash, s.seederKeyPrefix)
	if err != nil {
		return
	}
	resp.Complete = uint32(complete)
	incomplete, err := getPeersLength(s, infoHash, s.leecherKeyPrefix)
	if err != nil {
		return
	}
	resp.Incomplete = uint32(incomplete)
	return
}

func (s *peerStore) Stop() <-chan error {
	toReturn := make(chan error)
	close(s.closed)
	s.conn.Close()
	close(toReturn)
	return toReturn
}
