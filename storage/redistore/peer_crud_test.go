package redistore

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/chihaya/chihaya/bittorrent"
	"github.com/rafaeljusto/redigomock"
)

var testPeers = []struct {
	vals     bittorrent.Peer
	ih       bittorrent.InfoHash
	expected error
}{
	{bittorrent.Peer{
		ID:   bittorrent.PeerIDFromString("12345678912345678912"),
		IP:   net.IP("112.71.10.240"),
		Port: 7002},
		bittorrent.InfoHashFromString("12345678912345678912"), nil,
	},
	{bittorrent.Peer{
		ID:   bittorrent.PeerIDFromString("#!@#$%^$*()&*#$*~al:"),
		IP:   net.IP("10:71:10:1A:2B"),
		Port: 1111},
		bittorrent.InfoHashFromString("4:test3i:123:er123rt"), nil,
	}, {bittorrent.Peer{
		ID:   bittorrent.PeerIDFromString("////////////////////"),
		IP:   net.IP("192.168.0.2"),
		Port: 12356},
		bittorrent.InfoHashFromString("////////////////////"), nil,
	},
}

func TestAddAndCleanPeer(t *testing.T) {
	conn := redigomock.NewConn()
	ps := &peerStore{
		conn:         conn,
		closed:       make(chan struct{}),
		maxNumWant:   3,
		peerLifetime: 15,
		gcValidity:   1500000,
	}

	for _, tt := range testPeers {
		peer := fmt.Sprintf("%s%s", "seeder:", tt.ih)
		pk := newPeerKey(tt.vals)
		conn.Command("MULTI").Expect("OK")
		conn.Command("ZADD", peer, time.Now().Unix(), pk).Expect("QUEUED")
		conn.Command("EXPIRE", peer, ps.gcValidity).Expect("QUEUED")
		conn.Command("EXEC").Expect("1) OK\n2) OK")
		err := addAndCleanPeer(ps, tt.ih, "seeder", pk)
		if err != tt.expected {
			t.Error("addAndCleanPeer fail : ", err)
		}
	}
}
