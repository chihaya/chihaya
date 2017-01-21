package middleware

import (
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/chihaya/chihaya/bittorrent"
)

// nopHook is a Hook to measure the overhead of a no-operation Hook through
// benchmarks.
type nopHook struct{}

func (h *nopHook) HandleAnnounce(ctx context.Context, req *bittorrent.AnnounceRequest, resp *bittorrent.AnnounceResponse) (context.Context, error) {
	return ctx, nil
}

func (h *nopHook) HandleScrape(ctx context.Context, req *bittorrent.ScrapeRequest, resp *bittorrent.ScrapeResponse) (context.Context, error) {
	return ctx, nil
}

type hookList []Hook

func (hooks hookList) handleAnnounce(ctx context.Context, req *bittorrent.AnnounceRequest) (resp *bittorrent.AnnounceResponse, err error) {
	resp = &bittorrent.AnnounceResponse{
		Interval:    60,
		MinInterval: 60,
		Compact:     true,
	}

	for _, h := range []Hook(hooks) {
		if ctx, err = h.HandleAnnounce(ctx, req, resp); err != nil {
			return nil, err
		}
	}

	return resp, nil
}

func benchHookListV4(b *testing.B, hooks hookList) {
	req := &bittorrent.AnnounceRequest{Peer: bittorrent.Peer{IP: bittorrent.IP{IP: net.ParseIP("1.2.3.4"), AddressFamily: bittorrent.IPv4}}}
	benchHookList(b, hooks, req)
}

func benchHookListV6(b *testing.B, hooks hookList) {
	req := &bittorrent.AnnounceRequest{Peer: bittorrent.Peer{IP: bittorrent.IP{IP: net.ParseIP("fc00::0001"), AddressFamily: bittorrent.IPv6}}}
	benchHookList(b, hooks, req)
}

func benchHookList(b *testing.B, hooks hookList, req *bittorrent.AnnounceRequest) {
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := hooks.handleAnnounce(ctx, req)
		require.Nil(b, err)
		require.NotNil(b, resp)
	}
}

func BenchmarkHookOverhead(b *testing.B) {
	b.Run("none-v4", func(b *testing.B) {
		benchHookListV4(b, hookList{})
	})

	b.Run("none-v6", func(b *testing.B) {
		benchHookListV6(b, hookList{})
	})

	var nopHooks hookList
	for i := 1; i < 4; i++ {
		nopHooks = append(nopHooks, &nopHook{})
		b.Run(fmt.Sprintf("%dnop-v4", i), func(b *testing.B) {
			benchHookListV4(b, nopHooks)
		})
		b.Run(fmt.Sprintf("%dnop-v6", i), func(b *testing.B) {
			benchHookListV6(b, nopHooks)
		})
	}

	var sanHooks hookList
	for i := 1; i < 4; i++ {
		sanHooks = append(sanHooks, &sanitizationHook{maxNumWant: 50})
		b.Run(fmt.Sprintf("%dsanitation-v4", i), func(b *testing.B) {
			benchHookListV4(b, sanHooks)
		})
		b.Run(fmt.Sprintf("%dsanitation-v6", i), func(b *testing.B) {
			benchHookListV6(b, sanHooks)
		})
	}
}
