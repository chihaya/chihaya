package fixedpeers

import (
	"context"
	"encoding/hex"
	"net"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/chihaya/chihaya/bittorrent"
)

func TestAppendFixedPeer(t *testing.T) {
	conf := Config{
		FixedPeers: []string{"8.8.8.8:4040", "1.1.1.1:111"},
	}
	h, err := NewHook(conf)
	require.Nil(t, err)

	ctx := context.Background()
	req := &bittorrent.AnnounceRequest{}
	resp := &bittorrent.AnnounceResponse{}

	hashbytes, err := hex.DecodeString("3000000000000000000000000000000000000000")
	require.Nil(t, err)

	hashinfo := bittorrent.InfoHashFromBytes(hashbytes)

	req.InfoHash = hashinfo

	nctx, err := h.HandleAnnounce(ctx, req, resp)
	require.Equal(t, ctx, nctx)
	peers := []bittorrent.Peer{
		bittorrent.Peer{
			ID:   bittorrent.PeerID{0},
			Port: 4040,
			IP:   bittorrent.IP{net.ParseIP("8.8.8.8"), bittorrent.IPv4},
		},
		bittorrent.Peer{
			ID:   bittorrent.PeerID{0},
			Port: 111,
			IP:   bittorrent.IP{net.ParseIP("1.1.1.1"), bittorrent.IPv4},
		},
	}
	require.Equal(t, peers, resp.IPv4Peers)
}
