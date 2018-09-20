package clientapproval

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/chihaya/chihaya/bittorrent"
)

var cases = []struct {
	cfg      Config
	peerID   string
	approved bool
}{
	// Client ID is whitelisted
	{
		Config{
			Whitelist: []string{"010203"},
		},
		"01020304050607080900",
		true,
	},
	// Client ID is not whitelisted
	{
		Config{
			Whitelist: []string{"010203"},
		},
		"10203040506070809000",
		false,
	},
	// Client ID is not blacklisted
	{
		Config{
			Blacklist: []string{"010203"},
		},
		"00000000001234567890",
		true,
	},
	// Client ID is blacklisted
	{
		Config{
			Blacklist: []string{"123456"},
		},
		"12345678900000000000",
		false,
	},
}

func TestHandleAnnounce(t *testing.T) {
	for _, tt := range cases {
		t.Run(fmt.Sprintf("testing peerid %s", tt.peerID), func(t *testing.T) {
			h, err := NewHook(tt.cfg)
			require.Nil(t, err)

			ctx := context.Background()
			req := &bittorrent.AnnounceRequest{}
			resp := &bittorrent.AnnounceResponse{}

			peerid := bittorrent.PeerIDFromString(tt.peerID)

			req.Peer.ID = peerid

			nctx, err := h.HandleAnnounce(ctx, req, resp)
			require.Equal(t, ctx, nctx)
			if tt.approved == true {
				require.NotEqual(t, err, ErrClientUnapproved)
			} else {
				require.Equal(t, err, ErrClientUnapproved)
			}
		})
	}
}
