//Note: ip6 seperation into shards is unnecessary when using Redis(?)
package redis

import (
	"encoding/binary"
	"log"
	"time"

	"github.com/chihaya/chihaya/bittorrent"
	"github.com/chihaya/chihaya/storage"
	redigo "github.com/garyburd/redigo/redis"
)

var (
	prefixedSeeder  string
	prefixedLeecher string
)

type Config struct {
	Namespace    string        `yaml:"namespace"`
	Instance     int           `yaml:"instance"`
	MaxNumWant   int           `yaml:"max_numwant"`
	Host         string        `yaml:"host"`
	Port         string        `yaml:"port"`
	PeerLifetime time.Duration `yaml:"peer_liftetime"`
}

type peerStore struct {
	conn         redigo.Conn
	closed       chan struct{}
	maxNumWant   int
	peerLifetime time.Duration
	gcValidity   int
}

func New(cfg Config) (storage.PeerStore, error) {
	conn, err := redigo.Dial("tcp", cfg.Host+":"+cfg.Port)
	if err != nil {
		log.Fatal("Connection failed:" + err.Error())
		return nil, err
	}

	if cfg.Instance != 0 {
		conn.Do("SELECT", cfg.Instance)
	}
	prefixedSeeder = addNameSpace(cfg.Namespace, "seeder")
	prefixedLeecher = addNameSpace(cfg.Namespace, "leecher")

	ps := &peerStore{
		conn:         conn,
		closed:       make(chan struct{}),
		maxNumWant:   cfg.MaxNumWant,
		peerLifetime: cfg.PeerLifetime,
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

func addNameSpace(namespacePrefix string, command string) string {
	if namespacePrefix != "" {
		return namespacePrefix + ":" + command
	}
	return command
}

func (s *peerStore) PutSeeder(infoHash bittorrent.InfoHash, p bittorrent.Peer) error {
	panicIfClosed(s.closed)

	pk := newPeerKey(p)
	return addPeer(s, infoHash, prefixedSeeder, pk)
}

func (s *peerStore) DeleteSeeder(infoHash bittorrent.InfoHash, p bittorrent.Peer) error {
	panicIfClosed(s.closed)
	pk := newPeerKey(p)
	return removePeers(s, infoHash, prefixedSeeder, pk)
}

func (s *peerStore) PutLeecher(infoHash bittorrent.InfoHash, p bittorrent.Peer) error {
	panicIfClosed(s.closed)
	pk := newPeerKey(p)
	return addPeer(s, infoHash, prefixedLeecher, pk)

}

func (s *peerStore) DeleteLeecher(infoHash bittorrent.InfoHash, p bittorrent.Peer) error {
	panicIfClosed(s.closed)
	pk := newPeerKey(p)
	return removePeers(s, infoHash, prefixedLeecher, pk)

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

func (s *peerStore) AnnouncePeers(infoHash bittorrent.InfoHash, seeder bool, numWant int, announcer bittorrent.Peer) (peers []bittorrent.Peer, err error) {
	panicIfClosed(s.closed)
	if numWant > s.maxNumWant {
		numWant = s.maxNumWant
	}

	peers = []bittorrent.Peer{}
	if seeder {
		peers, err = getPeers(s, infoHash, prefixedLeecher, numWant, peers, bittorrent.Peer{})
		if err != nil {
			return nil, err
		}
	} else {
		peers, err = getPeers(s, infoHash, prefixedSeeder, numWant, peers, bittorrent.Peer{})
		if err != nil {
			return nil, err
		}
		if len(peers) < numWant {
			peers, err = getPeers(s, infoHash, prefixedLeecher, numWant, peers, announcer)
		}
	}
	return peers, nil
}

func (s *peerStore) ScrapeSwarm(infoHash bittorrent.InfoHash, v6 bool) (resp bittorrent.Scrape) {
	panicIfClosed(s.closed)
	complete, err := getPeersLength(s, infoHash, prefixedSeeder)
	if err != nil {
		return
	}
	resp.Complete = uint32(complete)
	incomplete, err := getPeersLength(s, infoHash, prefixedLeecher)
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
