package redis

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/RealImage/chihaya/bittorrent"
	"github.com/rafaeljusto/redigomock"
)

var testPeers = []struct {
	vals      bittorrent.Peer
	ih        bittorrent.InfoHash
	numPeers  int
	peers     []bittorrent.Peer
	announcer bittorrent.Peer
	expected  error
	rangeVals []interface{}
	expectedL int64
}{
	{bittorrent.Peer{
		ID:   bittorrent.PeerIDFromString("12345678912345678912"),
		IP:   net.IP("112.71.10.240"),
		Port: 7002},
		bittorrent.InfoHashFromString("12345678912345678912"), 0,
		[]bittorrent.Peer{},
		bittorrent.Peer{}, nil,
		[]interface{}{[]byte("12345678912345678912"),
			[]byte("abcdefgixxxxxxxxxxxxxxx"),
			[]byte("12354634ir78an0ob7151"),
			[]byte("000000000000000000000")},
		5,
	},
	{bittorrent.Peer{
		ID:   bittorrent.PeerIDFromString("#!@#$%^$*()&*#$*~al:"),
		IP:   net.IP("10:71:10:1A:2B"),
		Port: 1111},
		bittorrent.InfoHashFromString("4:test3i:123:er123rt"), 1,
		[]bittorrent.Peer{bittorrent.Peer{
			ID:   bittorrent.PeerIDFromString("totallydifferent1234"),
			IP:   net.IP("XX:71:10:1A:2X"),
			Port: 1234}},
		bittorrent.Peer{},
		nil,
		[]interface{}{[]byte("")},
		2,
	}, {bittorrent.Peer{
		ID:   bittorrent.PeerIDFromString("////////////////////"),
		IP:   net.IP("192.168.0.2"),
		Port: 12356},
		bittorrent.InfoHashFromString("////////////////////"), 0,
		[]bittorrent.Peer{},
		bittorrent.Peer{}, nil,
		[]interface{}{[]byte("")},
		1,
	},
}

func getPeerStore() (*redigomock.Conn, *peerStore) {
	conn := redigomock.NewConn()
	return conn, &peerStore{
		conn:         conn,
		closed:       make(chan struct{}),
		maxNumWant:   3,
		peerLifetime: 15,
		gcValidity:   1500000,
	}
}

func TestAddPeer(t *testing.T) {
	conn, ps := getPeerStore()
	for _, tp := range testPeers {
		peer := fmt.Sprintf("%s%s", "seeder:", tp.ih)
		pk := newPeerKey(tp.vals)
		conn.Command("MULTI").Expect("OK")
		conn.Command("ZADD", peer, time.Now().Unix(), pk).Expect("QUEUED")
		conn.Command("EXPIRE", peer, int(ps.peerLifetime.Seconds())).
			Expect("QUEUED")
		conn.Command("EXEC").Expect("1) OK\n2) OK")
		err := addPeer(ps, tp.ih, "seeder", pk)
		if err != tp.expected {
			t.Error("addPeer redis fail : ", err)
		}
	}
}

func TestRemPeers(t *testing.T) {
	conn, ps := getPeerStore()
	conn.Clear()
	for _, tp := range testPeers {
		peer := fmt.Sprintf("%s%s", "seeder:", tp.ih)
		pk := newPeerKey(tp.vals)
		conn.Command("ZREM", peer, pk).Expect("(integer) 1")
		err := removePeers(ps, tp.ih, "seeder", pk)
		if err != tp.expected {
			t.Error("remPeers redis fail:", err)
		}
	}
}

func TestGetPeers(t *testing.T) {
	conn, ps := getPeerStore()
	conn.Clear()
	for _, tp := range testPeers {
		peer := fmt.Sprintf("%s%s", "leecher:", tp.ih)
		conn.Command("ZREMRANGEBYSCORE", peer, "-inf",
			fmt.Sprintf("%s%d", "(", time.Now().Add(-ps.peerLifetime).Unix())).
			Expect("(integer 1)")
		conn.Command("ZRANGE", peer, 0, -1).Expect(tp.rangeVals)
		peers, err := getPeers(ps, tp.ih, "leecher",
			tp.numPeers, tp.peers, tp.announcer)
		if err != nil {
			t.Error("getPeers redis fail:", err)
		} else {
			if len(peers) != tp.numPeers {
				t.Error("getPeers logic fail : peer length issue")
			}
			for _, peerling := range peers {
				if peerling.ID == tp.announcer.ID {
					t.Error("getPeers logic fail : announcer not ignored")
				}
			}
		}
	}
}

func TestGetSetLength(t *testing.T) {
	conn, ps := getPeerStore()
	conn.Clear()
	for _, tp := range testPeers {
		peer := fmt.Sprintf("%s%s", "leecher:", tp.ih)
		conn.Command("ZCARD", peer).Expect(tp.expectedL)
		Actlen, err := getPeersLength(ps, tp.ih, "leecher")
		if err != nil {
			t.Error("getSEtLength redis fail: ", err)
		} else {
			if int64(Actlen) != tp.expectedL {
				t.Error("getSetLength logic fail: length mismatch")
			}

		}
	}
}
