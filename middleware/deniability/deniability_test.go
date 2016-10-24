package deniability

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/chihaya/chihaya/bittorrent"
)

type configTestDatum struct {
	cfg      Config
	expected error
}

var configTestData = []configTestDatum{
	{
		cfg:      Config{1.0, 50, "TEST", 1025, 65536},
		expected: nil,
	}, {
		cfg:      Config{1.0, 5, "", 1, 65536},
		expected: nil,
	}, {
		cfg:      Config{0, 5, "TEST", 1025, 65536},
		expected: ErrInvalidModifyResponseProbability,
	}, {
		cfg:      Config{1.0, 5, "TEST", 0, 65536},
		expected: ErrInvalidMinPort,
	}, {
		cfg:      Config{1.0, 5, "TEST", 1025, 1024},
		expected: ErrInvalidMaxPort,
	}, {
		cfg:      Config{1.0, 5, "TEST", 1025, 100000},
		expected: ErrInvalidMaxPort,
	}, {
		cfg:      Config{1.0, 5, "01234567890123456789_", 1025, 65536},
		expected: ErrInvalidPrefix,
	},
}

func TestCheckConfig(t *testing.T) {
	for _, d := range configTestData {
		got := checkConfig(d.cfg)
		require.Equal(t, d.expected, got, "", fmt.Sprintf("%+v", d.cfg))
	}
}

func TestNew(t *testing.T) {
	hook, err := New(configTestData[0].cfg)
	require.Nil(t, err)
	require.NotNil(t, hook)
}

func TestHook_HandleAnnounce(t *testing.T) {
	knownPeerID := bittorrent.PeerIDFromString("00000000001111111111")
	req := &bittorrent.AnnounceRequest{NumWant: 50, Peer: bittorrent.Peer{IP: bittorrent.IP{IP: net.IP([]byte{1, 2, 3, 4}), AddressFamily: bittorrent.IPv4}}}
	resp := &bittorrent.AnnounceResponse{IPv4Peers: []bittorrent.Peer{{ID: knownPeerID, IP: bittorrent.IP{IP: net.IP([]byte{2, 3, 4, 5}), AddressFamily: bittorrent.IPv4}}}}
	cfg := configTestData[0].cfg

	hook, err := New(cfg)
	require.Nil(t, err)
	ctx := context.Background()
	nCtx, err := hook.HandleAnnounce(ctx, req, resp)
	require.Nil(t, err)
	require.Equal(t, ctx, nCtx)

	require.True(t, len(resp.IPv4Peers) > 1)
	for _, peer := range resp.IPv4Peers {
		if bytes.Equal(peer.ID[:], knownPeerID[:]) {
			continue
		}

		require.Equal(t, len(req.Peer.IP.IP), len(peer.IP.IP))
		require.Equal(t, cfg.Prefix, string(peer.ID[:len(cfg.Prefix)]))
		require.True(t, int(peer.Port) < cfg.MaxPort)
		require.True(t, peer.Port >= cfg.MinPort)
	}
}

// manyPeers returns 50 Peers
func makePeers() []bittorrent.Peer {
	return []bittorrent.Peer{{}, {}, {}, {}, {}, {}, {}, {}, {}, {},
		{}, {}, {}, {}, {}, {}, {}, {}, {}, {},
		{}, {}, {}, {}, {}, {}, {}, {}, {}, {},
		{}, {}, {}, {}, {}, {}, {}, {}, {}, {},
		{}, {}, {}, {}, {}, {}, {}, {}, {}, {}}
}

func makeBenchmark(b *testing.B, cfg Config, v6, manyPeers bool) func(b *testing.B) {
	req := &bittorrent.AnnounceRequest{NumWant: 50, Peer: bittorrent.Peer{ID: bittorrent.PeerIDFromString("12345678901234567890"), IP: bittorrent.IP{IP: net.IP([]byte{1, 2, 3, 4}), AddressFamily: bittorrent.IPv4}}}
	resp := &bittorrent.AnnounceResponse{}
	if v6 {
		req.Peer.IP.IP = net.ParseIP("2001:db8::68")
		req.Peer.IP.AddressFamily = bittorrent.IPv6
	}
	if manyPeers {
		if v6 {
			resp.IPv6Peers = makePeers()
		} else {
			resp.IPv4Peers = makePeers()
		}
	}

	hook, err := New(cfg)
	require.Nil(b, err)
	ctx := context.Background()

	return func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			hook.HandleAnnounce(ctx, req, resp)
			if !manyPeers {
				// Reset this, fills up otherwise.
				// We can not set them to nil, the hook ignores
				// empty responses.
				resp.IPv4Peers = []bittorrent.Peer{{}}
				resp.IPv6Peers = []bittorrent.Peer{{}}
			}
		}
	}
}

func BenchmarkHook_HandleAnnounce(b *testing.B) {
	cfg := configTestData[0].cfg
	chances := []float32{0.1, 0.25, 0.5, 1}
	max := []int{1, 5, 10, 50}
	bools := []bool{false, true}
	for _, c := range chances {
		for _, m := range max {
			for _, ipv6 := range bools {
				for _, manyPeers := range bools {
					// If we have manyPeers the hook will replace instead of inserting.
					cfg.MaxRandomPeers = m
					cfg.ModifyResponseProbability = c
					b.Run(fmt.Sprintf("modify=%f, max=%d, ipv6=%t, replace=%t", c, m, ipv6, manyPeers),
						makeBenchmark(b, cfg, ipv6, manyPeers))
				}
			}
		}
	}
}
