package varinterval

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/chihaya/chihaya/bittorrent"
)

var configTests = []struct {
	cfg      Config
	expected error
}{
	{
		cfg:      Config{0.5, 60, true},
		expected: nil,
	}, {
		cfg:      Config{1.0, 60, true},
		expected: nil,
	}, {
		cfg:      Config{0.0, 60, true},
		expected: ErrInvalidModifyResponseProbability,
	}, {
		cfg:      Config{1.1, 60, true},
		expected: ErrInvalidModifyResponseProbability,
	}, {
		cfg:      Config{0.5, 0, true},
		expected: ErrInvalidMaxIncreaseDelta,
	}, {
		cfg:      Config{0.5, -10, true},
		expected: ErrInvalidMaxIncreaseDelta,
	},
}

func TestCheckConfig(t *testing.T) {
	for _, tt := range configTests {
		t.Run(fmt.Sprintf("%#v", tt.cfg), func(t *testing.T) {
			got := checkConfig(tt.cfg)
			require.Equal(t, tt.expected, got, "", tt.cfg)
		})
	}
}

func TestHandleAnnounce(t *testing.T) {
	h, err := NewHook(Config{1.0, 10, true})
	require.Nil(t, err)
	require.NotNil(t, h)

	ctx := context.Background()
	req := &bittorrent.AnnounceRequest{}
	resp := &bittorrent.AnnounceResponse{}

	nCtx, err := h.HandleAnnounce(ctx, req, resp)
	require.Nil(t, err)
	require.Equal(t, ctx, nCtx)
	require.True(t, resp.Interval > 0, "interval should have been increased")
	require.True(t, resp.MinInterval > 0, "min_interval should have been increased")
}
