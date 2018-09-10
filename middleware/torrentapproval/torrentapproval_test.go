package torrentapproval

import (
	"context"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/chihaya/chihaya/bittorrent"
	"github.com/stretchr/testify/require"
)

var cases = []struct {
	cfg      Config
	ih       string
	approved bool
}{
	// Infohash is whitelisted
	{
		Config{
			Whitelist: []string{"3532cf2d327fad8448c075b4cb42c8136964a435"},
		},
		"3532cf2d327fad8448c075b4cb42c8136964a435",
		true,
	},
	// Infohash is not whitelisted
	{
		Config{
			Whitelist: []string{"3532cf2d327fad8448c075b4cb42c8136964a435"},
		},
		"4532cf2d327fad8448c075b4cb42c8136964a435",
		false,
	},
	// Infohash is not blacklisted
	{
		Config{
			Blacklist: []string{"3532cf2d327fad8448c075b4cb42c8136964a435"},
		},
		"4532cf2d327fad8448c075b4cb42c8136964a435",
		true,
	},
	// Infohash is blacklisted
	{
		Config{
			Blacklist: []string{"3532cf2d327fad8448c075b4cb42c8136964a435"},
		},
		"3532cf2d327fad8448c075b4cb42c8136964a435",
		false,
	},
}

func TestHandleAnnounce(t *testing.T) {
	for _, tt := range cases {
		t.Run(fmt.Sprintf("testing hash %s", tt.ih), func(t *testing.T) {
			h, err := NewHook(tt.cfg)
			require.Nil(t, err)

			ctx := context.Background()
			req := &bittorrent.AnnounceRequest{}
			resp := &bittorrent.AnnounceResponse{}

			hashbytes, err := hex.DecodeString(tt.ih)
			require.Nil(t, err)

			hashinfo := bittorrent.InfoHashFromBytes(hashbytes)

			req.InfoHash = hashinfo

			nctx, err := h.HandleAnnounce(ctx, req, resp)
			require.Equal(t, ctx, nctx)
			if tt.approved == true {
				require.Nil(t, err)
			} else {
				require.NotNil(t, err)
			}
		})
	}
}
