package bittorrent

import (
	"fmt"
	"net/netip"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	b        = []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	expected = "0102030405060708090a0b0c0d0e0f1011121314"
)

var peerStringTestCases = []struct {
	input    Peer
	expected string
}{
	{
		input: Peer{
			ID:       PeerIDFromBytes(b),
			AddrPort: netip.MustParseAddrPort("10.11.12.1:1234"),
		},
		expected: fmt.Sprintf("%s@[10.11.12.1]:1234", expected),
	},
	{
		input: Peer{
			ID:       PeerIDFromBytes(b),
			AddrPort: netip.MustParseAddrPort("[2001:db8::ff00:42:8329]:1234"),
		},
		expected: fmt.Sprintf("%s@[2001:db8::ff00:42:8329]:1234", expected),
	},
}

func TestPeerID_String(t *testing.T) {
	s := PeerIDFromBytes(b).String()
	require.Equal(t, expected, s)
}

func TestInfoHash_String(t *testing.T) {
	s := InfoHashFromBytes(b).String()
	require.Equal(t, expected, s)
}

func TestPeer_String(t *testing.T) {
	for _, c := range peerStringTestCases {
		got := c.input.String()
		require.Equal(t, c.expected, got)
	}
}
