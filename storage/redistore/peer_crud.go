package redistore

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/chihaya/chihaya/bittorrent"
	"github.com/garyburd/redigo/redis"
)

func decodePeerKey(pk string) bittorrent.Peer {
	return bittorrent.Peer{
		ID:   bittorrent.PeerIDFromString(string(pk[:20])),
		Port: binary.BigEndian.Uint16([]byte(pk[20:22])),
		IP:   net.IP(pk[22:]),
	}
}

func addAndCleanPeer(s *peerStore, infoHash bittorrent.InfoHash, peerType string, pk serializedPeer) error {
	nameSpacedKey := addNameSpace(fmt.Sprintf("%s%s", peerType+":", infoHash))
	s.conn.Send("WATCH")
	s.conn.Send("ZADD", nameSpacedKey, time.Now().Unix(), pk)
	s.conn.Send("ZREMRANGEBYSCORE", nameSpacedKey,
		"-inf", fmt.Sprintf("%s%s", "(", time.Now().Add(-s.peerLifetime)))
	s.conn.Send("EXPIRE", nameSpacedKey, s.gcValidity)
	_, err := s.conn.Do("EXEC")
	if err != nil {
		return err
	}
	return nil
}

func remPeers(s *peerStore, infoHash bittorrent.InfoHash, peerType string, pk serializedPeer) error {
	_, err := s.conn.Do("ZREM",
		addNameSpace(fmt.Sprintf("%s%s", peerType+":", infoHash)),
		pk)
	if err != nil {
		return err
	}
	return nil
}

func getPeers(s *peerStore, infoHash bittorrent.InfoHash, peerType string, numWant int, peers []bittorrent.Peer, announcer bittorrent.Peer) ([]bittorrent.Peer, error) {
	peerList, err := redis.Strings(s.conn.Do("ZRANGE",
		addNameSpace(fmt.Sprintf("%s%s", peerType+":", infoHash)), 0, -1))
	if err != nil {
		return nil, err
	}
	for _, p := range peerList {
		if numWant == len(peers) {
			break
		}
		decodedPeer := decodePeerKey(p)
		if decodedPeer.Equal(announcer) {
			continue
		}
		peers = append(peers, decodedPeer)
	}
	return peers, nil
}
