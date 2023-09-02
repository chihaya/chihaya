package storage

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/chihaya/chihaya/bittorrent"
)

// PeerEqualityFunc is the boolean function to use to check two Peers for
// equality.
// Depending on the implementation of the PeerStore, this can be changed to
// use (Peer).EqualEndpoint instead.
var PeerEqualityFunc = func(p1, p2 bittorrent.Peer) bool { return p1.Equal(p2) }

// TestPeerStore tests a PeerStore implementation against the interface.
func TestPeerStore(t *testing.T, p PeerStore) {
	testData := []struct {
		ih   bittorrent.InfoHash
		peer bittorrent.Peer
	}{
		{
			bittorrent.InfoHashFromString("00000000000000000001"),
			bittorrent.Peer{ID: bittorrent.PeerIDFromString("00000000000000000001"), Port: 1, IP: bittorrent.IP{IP: net.ParseIP("1.1.1.1").To4(), AddressFamily: bittorrent.IPv4}},
		},
		{
			bittorrent.InfoHashFromString("00000000000000000002"),
			bittorrent.Peer{ID: bittorrent.PeerIDFromString("00000000000000000002"), Port: 2, IP: bittorrent.IP{IP: net.ParseIP("abab::0001"), AddressFamily: bittorrent.IPv6}},
		},
	}

	v4Peer := bittorrent.Peer{ID: bittorrent.PeerIDFromString("99999999999999999994"), IP: bittorrent.IP{IP: net.ParseIP("99.99.99.99").To4(), AddressFamily: bittorrent.IPv4}, Port: 9994}
	v6Peer := bittorrent.Peer{ID: bittorrent.PeerIDFromString("99999999999999999996"), IP: bittorrent.IP{IP: net.ParseIP("fc00::0001"), AddressFamily: bittorrent.IPv6}, Port: 9996}

	for _, c := range testData {
		peer := v4Peer
		if c.peer.IP.AddressFamily == bittorrent.IPv6 {
			peer = v6Peer
		}

		// Test ErrDNE for non-existent swarms.
		err := p.DeleteLeecher(c.ih, c.peer)
		require.Equal(t, ErrResourceDoesNotExist, err)

		err = p.DeleteSeeder(c.ih, c.peer)
		require.Equal(t, ErrResourceDoesNotExist, err)

		_, err = p.AnnouncePeers(c.ih, false, 50, peer, nil)
		require.Equal(t, ErrResourceDoesNotExist, err)

		// Test empty scrape response for non-existent swarms.
		scrape := p.ScrapeSwarm(c.ih, c.peer.IP.AddressFamily)
		require.Equal(t, uint32(0), scrape.Complete)
		require.Equal(t, uint32(0), scrape.Incomplete)
		require.Equal(t, uint32(0), scrape.Snatches)

		// Insert dummy Peer to keep swarm active
		// Has the same address family as c.peer
		err = p.PutLeecher(c.ih, peer)
		require.Nil(t, err)

		// Test ErrDNE for non-existent seeder.
		err = p.DeleteSeeder(c.ih, peer)
		require.Equal(t, ErrResourceDoesNotExist, err)

		// Test PutLeecher -> Announce -> DeleteLeecher -> Announce

		err = p.PutLeecher(c.ih, c.peer)
		require.Nil(t, err)

		peers, err := p.AnnouncePeers(c.ih, true, 50, peer, nil)
		require.Nil(t, err)
		require.True(t, containsPeer(peers, c.peer))

		// non-seeder announce should still return the leecher
		peers, err = p.AnnouncePeers(c.ih, false, 50, peer, nil)
		require.Nil(t, err)
		require.True(t, containsPeer(peers, c.peer))

		scrape = p.ScrapeSwarm(c.ih, c.peer.IP.AddressFamily)
		require.Equal(t, uint32(2), scrape.Incomplete)
		require.Equal(t, uint32(0), scrape.Complete)

		err = p.DeleteLeecher(c.ih, c.peer)
		require.Nil(t, err)

		peers, err = p.AnnouncePeers(c.ih, true, 50, peer, nil)
		require.Nil(t, err)
		require.False(t, containsPeer(peers, c.peer))

		// Test PutSeeder -> Announce -> DeleteSeeder -> Announce

		err = p.PutSeeder(c.ih, c.peer)
		require.Nil(t, err)

		// Should be leecher to see the seeder
		peers, err = p.AnnouncePeers(c.ih, false, 50, peer, nil)
		require.Nil(t, err)
		require.True(t, containsPeer(peers, c.peer))

		scrape = p.ScrapeSwarm(c.ih, c.peer.IP.AddressFamily)
		require.Equal(t, uint32(1), scrape.Incomplete)
		require.Equal(t, uint32(1), scrape.Complete)

		err = p.DeleteSeeder(c.ih, c.peer)
		require.Nil(t, err)

		peers, err = p.AnnouncePeers(c.ih, false, 50, peer, nil)
		require.Nil(t, err)
		require.False(t, containsPeer(peers, c.peer))

		// Test PutLeecher -> Graduate -> Announce -> DeleteLeecher -> Announce

		err = p.PutLeecher(c.ih, c.peer)
		require.Nil(t, err)

		err = p.GraduateLeecher(c.ih, c.peer)
		require.Nil(t, err)

		// Has to be leecher to see the graduated seeder
		peers, err = p.AnnouncePeers(c.ih, false, 50, peer, nil)
		require.Nil(t, err)
		require.True(t, containsPeer(peers, c.peer))

		// Deleting the Peer as a Leecher should have no effect
		err = p.DeleteLeecher(c.ih, c.peer)
		require.Equal(t, ErrResourceDoesNotExist, err)

		// Verify it's still there
		peers, err = p.AnnouncePeers(c.ih, false, 50, peer, nil)
		require.Nil(t, err)
		require.True(t, containsPeer(peers, c.peer))

		// Clean up

		err = p.DeleteLeecher(c.ih, peer)
		require.Nil(t, err)

		// Test ErrDNE for missing leecher
		err = p.DeleteLeecher(c.ih, peer)
		require.Equal(t, ErrResourceDoesNotExist, err)

		err = p.DeleteSeeder(c.ih, c.peer)
		require.Nil(t, err)

		err = p.DeleteSeeder(c.ih, c.peer)
		require.Equal(t, ErrResourceDoesNotExist, err)
	}

	e := p.Stop()
	require.Nil(t, <-e)
}

func containsPeer(peers []bittorrent.Peer, p bittorrent.Peer) bool {
	for _, peer := range peers {
		if PeerEqualityFunc(peer, p) {
			return true
		}
	}
	return false
}
