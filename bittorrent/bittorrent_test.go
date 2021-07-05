package bittorrent

import (
	"testing"

	"github.com/stretchr/testify/require"
	"inet.af/netaddr"
)

var peerIDTable = []struct {
	name   string
	peerID [20]byte
	raw    string
	hex    string
}{
	{"empty", [20]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, "\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00", "0000000000000000000000000000000000000000"},
	{"real", [20]byte{0x41, 0x5a, 0x32, 0x35, 0x30, 0x30, 0x42, 0x54, 0x65, 0x59, 0x55, 0x7a, 0x79, 0x61, 0x62, 0x41, 0x66, 0x6f, 0x36, 0x55}, "\x41\x5a\x32\x35\x30\x30\x42\x54\x65\x59\x55\x7a\x79\x61\x62\x41\x66\x6f\x36\x55", "415a3235303042546559557a79616241666f3655"},
}

func TestPeerIDString(t *testing.T) {
	for _, tt := range peerIDTable {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.hex, PeerID(tt.peerID).String())
		})
	}
}

func TestPeerIDFromRawString(t *testing.T) {
	for _, tt := range peerIDTable {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.peerID, [20]byte(PeerIDFromRawString(tt.raw)))
		})
	}
}

func TestPeerIDRawString(t *testing.T) {
	for _, tt := range peerIDTable {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.raw, PeerID(tt.peerID).RawString())
		})
	}
}

func TestPeerIDFromHexString(t *testing.T) {
	for _, tt := range peerIDTable {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.peerID, [20]byte(PeerIDFromHexString(tt.hex)))
		})
	}
}

var peerStringTable = []struct {
	name       string
	peer       Peer
	peerRawStr string
}{
	{
		name: "ipv4",
		peer: Peer{
			ID:     PeerID([20]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}),
			IPPort: netaddr.IPPortFrom(netaddr.MustParseIP("10.11.12.1"), 1234),
		},
		peerRawStr: "\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f\x10\x11\x12\x13\x14\x04\xd2\x0a\x0b\x0c\x01",
	},
	{
		name: "ipv6",
		peer: Peer{
			ID:     PeerID([20]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}),
			IPPort: netaddr.IPPortFrom(netaddr.MustParseIP("2001:db8::ff00:42:8329"), 1234),
		},
		peerRawStr: "\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f\x10\x11\x12\x13\x14\x04\xd2\x20\x01\x0d\xb8\x00\x00\x00\x00\x00\x00\xff\x00\x00\x42\x83\x29",
	},
}

func TestPeerString(t *testing.T) {
	for _, tt := range peerStringTable {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.peerRawStr, tt.peer.RawString())
		})
	}
}

func TestMustParsePeer(t *testing.T) {
	for _, tt := range peerStringTable {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.peer, PeerFromRawString(tt.peerRawStr))
		})
	}
}

func TestClientID(t *testing.T) {
	clientTable := []struct{ peerID, clientID string }{
		{"-AZ3034-6wfG2wk6wWLc", "AZ3034"},
		{"-AZ3042-6ozMq5q6Q3NX", "AZ3042"},
		{"-BS5820-oy4La2MWGEFj", "BS5820"},
		{"-AR6360-6oZyyMWoOOBe", "AR6360"},
		{"-AG2083-s1hiF8vGAAg0", "AG2083"},
		{"-AG3003-lEl2Mm4NEO4n", "AG3003"},
		{"-MR1100-00HS~T7*65rm", "MR1100"},
		{"-LK0140-ATIV~nbEQAMr", "LK0140"},
		{"-KT2210-347143496631", "KT2210"},
		{"-TR0960-6ep6svaa61r4", "TR0960"},
		{"-XX1150-dv220cotgj4d", "XX1150"},
		{"-AZ2504-192gwethivju", "AZ2504"},
		{"-KT4310-3L4UvarKuqIu", "KT4310"},
		{"-AZ2060-0xJQ02d4309O", "AZ2060"},
		{"-BD0300-2nkdf08Jd890", "BD0300"},
		{"-A~0010-a9mn9DFkj39J", "A~0010"},
		{"-UT2300-MNu93JKnm930", "UT2300"},
		{"-UT2300-KT4310KT4301", "UT2300"},

		{"T03A0----f089kjsdf6e", "T03A0-"},
		{"S58B-----nKl34GoNb75", "S58B--"},
		{"M4-4-0--9aa757Efd5Bl", "M4-4-0"},

		{"AZ2500BTeYUzyabAfo6U", "AZ2500"}, // BitTyrant
		{"exbc0JdSklm834kj9Udf", "exbc0J"}, // Old BitComet
		{"FUTB0L84j542mVc84jkd", "FUTB0L"}, // Alt BitComet
		{"XBT054d-8602Jn83NnF9", "XBT054"}, // XBT
		{"OP1011affbecbfabeefb", "OP1011"}, // Opera
		{"-ML2.7.2-kgjjfkd9762", "ML2.7."}, // MLDonkey
		{"-BOWA0C-SDLFJWEIORNM", "BOWA0C"}, // Bits on Wheels
		{"Q1-0-0--dsn34DFn9083", "Q1-0-0"}, // Queen Bee
		{"Q1-10-0-Yoiumn39BDfO", "Q1-10-"}, // Queen Bee Alt
		{"346------SDFknl33408", "346---"}, // TorreTopia
		{"QVOD0054ABFFEDCCDEDB", "QVOD00"}, // Qvod
	}

	for _, tt := range clientTable {
		t.Run(tt.peerID, func(t *testing.T) {
			var clientID ClientID
			copy(clientID[:], []byte(tt.clientID))
			require.Equal(t, clientID, PeerIDFromRawString(tt.peerID).ClientID())
		})
	}
}
