package redistore

import (
	"fmt"
	"time"

	"github.com/chihaya/chihaya/bittorrent"
)

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
