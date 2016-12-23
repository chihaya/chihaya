package redis

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/chihaya/chihaya/bittorrent"
	redigo "github.com/garyburd/redigo/redis"
)

func decodePeerKey(pk string) bittorrent.Peer {
	return bittorrent.Peer{
		ID:   bittorrent.PeerIDFromString(string(pk[:20])),
		Port: binary.BigEndian.Uint16([]byte(pk[20:22])),
		IP:   net.IP(pk[22:]),
	}
}

func addPeer(s *peerStore, infoHash bittorrent.InfoHash, peerType string, pk serializedPeer) error {
	Key := fmt.Sprintf("%s%s", peerType+":", infoHash)
	s.conn.Send("MULTI")
	s.conn.Send("ZADD", Key, time.Now().Unix(), pk)
	s.conn.Send("EXPIRE", Key, int(s.peerLifetime.Seconds()))
	_, err := s.conn.Do("EXEC")
	if err != nil {
		return err
	}
	return nil
}

func removePeers(s *peerStore, infoHash bittorrent.InfoHash, peerType string, pk serializedPeer) error {
	_, err := s.conn.Do("ZREM",
		fmt.Sprintf("%s%s", peerType+":", infoHash), pk)
	if err != nil {
		return err
	}
	return nil
}

func getPeers(s *peerStore, infoHash bittorrent.InfoHash, peerType string, numWant int, peers []bittorrent.Peer, excludePeers bittorrent.Peer) ([]bittorrent.Peer, error) {
	Key := fmt.Sprintf("%s%s", peerType+":", infoHash)
	_, err := s.conn.Do("ZREMRANGEBYSCORE", Key,
		"-inf", fmt.Sprintf("%s%d", "(", time.Now().Add(-s.peerLifetime).Unix()))
	if err != nil {
		return nil, err
	}
	peerList, err := redigo.Strings(s.conn.Do("ZRANGE",
		Key, 0, -1))
	if err != nil {
		return nil, err
	}
	for _, p := range peerList {
		if numWant == len(peers) {
			break
		}
		decodedPeer := decodePeerKey(p)
		if decodedPeer.Equal(excludePeers) {
			continue
		}
		peers = append(peers, decodedPeer)
	}
	return peers, nil
}

func getPeersLength(s *peerStore, infoHash bittorrent.InfoHash, peerType string) (int, error) {
	return redigo.Int(s.conn.Do("ZCARD",
		fmt.Sprintf("%s%s", peerType+":", infoHash)))
}
