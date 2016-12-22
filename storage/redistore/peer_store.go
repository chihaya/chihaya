//Note: ip6 seperation into shards is unnecessary when using Redis(?)
package redistore

import (
	"encoding/binary"
	"fmt"
	"log"
	"time"

	"github.com/chihaya/chihaya/bittorrent"
	"github.com/chihaya/chihaya/storage"
	"github.com/garyburd/redigo/redis"
)

var namespacePrefix string

type Config struct {
	Namespace bool
	//use as namespace prefix if Namespace = yes or as instance no.
	Cntrl                     string
	MaxNumWant                int
	Host                      string
	Port                      string
	GarbageCollectionInterval int
	PeerLifetime              time.Duration
}

type peerStore struct {
	conn         redis.Conn
	closed       chan struct{}
	maxNumWant   int
	peerLifetime time.Duration
	gcValidity   int
}

func New(cfg Config) (storage.PeerStore, error) {
	conn, err := redis.Dial("tcp", cfg.Host+":"+cfg.Port)
	if err != nil {
		log.Fatal("Connection failed:" + err.Error())
		return nil, err
	}

	if !cfg.Namespace {
		conn.Do("SELECT", cfg.Cntrl)
	} else {
		namespacePrefix = cfg.Cntrl
	}

	ps := &peerStore{
		conn:         conn,
		closed:       make(chan struct{}),
		maxNumWant:   cfg.MaxNumWant,
		peerLifetime: cfg.PeerLifetime,
		gcValidity:   cfg.GarbageCollectionInterval,
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
		panic("attempted to interact with stopped memory store")
	default:
	}
}

func addNameSpace(command string) string {
	return namespacePrefix + ":" + command
}

func (s *peerStore) PutSeeder(infoHash bittorrent.InfoHash, p bittorrent.Peer) error {
	panicIfClosed(s.closed)

	pk := newPeerKey(p)
	return addAndCleanPeer(s, infoHash, "seeder", pk)
}

func (s *peerStore) DeleteSeeder(infoHash bittorrent.InfoHash, p bittorrent.Peer) error {
	panicIfClosed(s.closed)
	pk := newPeerKey(p)
	return remPeers(s, infoHash, "seeder", pk)
}

func (s *peerStore) PutLeecher(infoHash bittorrent.InfoHash, p bittorrent.Peer) error {
	panicIfClosed(s.closed)
	pk := newPeerKey(p)
	return addAndCleanPeer(s, infoHash, "leecher", pk)

}

func (s *peerStore) DeleteLeecher(infoHash bittorrent.InfoHash, p bittorrent.Peer) error {
	panicIfClosed(s.closed)
	pk := newPeerKey(p)
	return remPeers(s, infoHash, "leecher", pk)

}

func (s *peerStore) GraduateLeecher(infoHash bittorrent.InfoHash, p bittorrent.Peer) error {
	panicIfClosed(s.closed)
	err := s.DeleteLeecher(infoHash, p)
	if err != nil {
		return err
	}
	err = s.PutSeeder(infoHash, p)
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
		peers, err = getPeers(s, infoHash, "leecher", numWant, peers, bittorrent.Peer{})
		if err != nil {
			return nil, err
		}
	} else {
		peers, err = getPeers(s, infoHash, "seeder", numWant, peers, bittorrent.Peer{})
		if err != nil {
			return nil, err
		}
		if numWant > len(peers) {
			peers, err = getPeers(s, infoHash, "leecher", numWant, peers, announcer)
		}
	}
	return peers, nil
}

func (s *peerStore) ScrapeSwarm(infoHash bittorrent.InfoHash, v6 bool) (resp bittorrent.Scrape) {
	panicIfClosed(s.closed)
	complete, err := redis.Int(s.conn.Do("ZCARD",
		addNameSpace(fmt.Sprintf("%s%s", "seeder:", infoHash))))
	if err != nil {
		return
	}
	resp.Complete = uint32(complete)
	incomplete, err := redis.Int(s.conn.Do("ZCARD",
		addNameSpace(fmt.Sprintf("%s%s", "leecher:", infoHash))))
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
