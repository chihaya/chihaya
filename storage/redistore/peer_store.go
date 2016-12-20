package redistore

import (
	"encoding/binary"
	"log"
	"net"
	"time"

	"github.com/chihaya/chihaya/bittorrent"
	"github.com/chihaya/chihaya/storage"
	"github.com/garyburd/redigo/redis"
)

var namespacePrefix string

type Config struct {
	PeerLifetime int
	Namespace    bool
	//use as namespace prefix if Namespace = yes or as instance no.
	Cntrl      string
	MaxNumWant int
	Host       string
	Port       string
}

type Connecter interface {
	New() (redis.Conn, error)
}

type peerStore struct {
	conn       redis.Conn
	closed     chan struct{}
	maxNumWant int
}

func (cfg Config) New() (storage.PeerStore, error) {
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
		conn:       conn,
		closed:     make(chan struct{}),
		maxNumWant: cfg.MaxNumWant,
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

func decodePeerKey(pk serializedPeer) bittorrent.Peer {
	return bittorrent.Peer{
		ID:   bittorrent.PeerIDFromString(string(pk[:20])),
		Port: binary.BigEndian.Uint16([]byte(pk[20:22])),
		IP:   net.IP(pk[22:]),
	}
}

func panicIfClosed(closed <-chan struct{}) {
	select {
	case <-closed:
		panic("attempted to interact with stopped memory store")
	default:
	}
}

func (s *peerStore) PutSeeder(infoHash bittorrent.InfoHash, p bittorrent.Peer) error {
	panicIfClosed(s.closed)

	pk := newPeerKey(p)
	_, err := s.conn.Do("ZADD", "seeder:"+string(infoHash[:20]),
		time.Now().Unix(), pk)
	if err != nil {
		return err
	}
	return nil
}

func (s *peerStore) DeleteSeeder(infoHash bittorrent.InfoHash, p bittorrent.Peer) error {
	panicIfClosed(s.closed)
	return nil
}

func (s *peerStore) PutLeecher(infoHash bittorrent.InfoHash, p bittorrent.Peer) error {
	panicIfClosed(s.closed)
	pk := newPeerKey(p)
	_, err := s.conn.Do("ZADD", "leecher:"+string(infoHash[:20]),
		time.Now().Unix(), pk)
	if err != nil {
		return err
	}
	return nil
}

func (s *peerStore) DeleteLeecher(infoHash bittorrent.InfoHash, p bittorrent.Peer) error {
	panicIfClosed(s.closed)
	return nil
}

func (s *peerStore) GraduateLeecher(infoHash bittorrent.InfoHash, p bittorrent.Peer) error {
	panicIfClosed(s.closed)
	return nil
}

func (s *peerStore) AnnouncePeers(infoHash bittorrent.InfoHash, seeder bool, numWant int, p bittorrent.Peer) (peers []bittorrent.Peer, err error) {
	panicIfClosed(s.closed)
	return peers, nil
}

func (s *peerStore) ScrapeSwarm(infoHash bittorrent.InfoHash, v6 bool) bittorrent.Scrape {
	panicIfClosed(s.closed)
	return bittorrent.Scrape{}
}

func (s *peerStore) Stop() <-chan error {
	return nil
}
