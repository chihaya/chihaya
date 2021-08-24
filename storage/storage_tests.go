package storage

import (
	"testing"

	"github.com/stretchr/testify/require"
	"inet.af/netaddr"

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
			bittorrent.Peer{
				ID:     bittorrent.PeerIDFromRawString("00000000000000000001"),
				IPPort: netaddr.MustParseIPPort("1.1.1.1:1"),
			},
		},
		{
			bittorrent.InfoHashFromString("00000000000000000002"),
			bittorrent.Peer{
				ID:     bittorrent.PeerIDFromRawString("00000000000000000002"),
				IPPort: netaddr.MustParseIPPort("[abab::0001]:2"),
			},
		},
	}

	v4Peer := bittorrent.Peer{ID: bittorrent.PeerIDFromRawString("99999999999999999994"), IPPort: netaddr.MustParseIPPort("99.99.99.99:9994")}
	v6Peer := bittorrent.Peer{ID: bittorrent.PeerIDFromRawString("99999999999999999996"), IPPort: netaddr.MustParseIPPort("[fc00::0001]:9996")}

	for _, c := range testData {
		peer := v4Peer
		if c.peer.IPPort.IP().Is6() {
			peer = v6Peer
		}

		// Test ErrDNE for non-existent swarms.
		err := p.DeleteLeecher(c.ih, c.peer)
		require.Equal(t, ErrResourceDoesNotExist, err)

		err = p.DeleteSeeder(c.ih, c.peer)
		require.Equal(t, ErrResourceDoesNotExist, err)

		_, err = p.AnnouncePeers(c.ih, false, 50, peer)
		require.Equal(t, ErrResourceDoesNotExist, err)

		// Test empty scrape response for non-existent swarms.
		scrape := p.ScrapeSwarm(c.ih, c.peer.IPPort.IP())
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

		peers, err := p.AnnouncePeers(c.ih, true, 50, peer)
		require.Nil(t, err)
		require.True(t, containsPeer(peers, c.peer))

		// non-seeder announce should still return the leecher
		peers, err = p.AnnouncePeers(c.ih, false, 50, peer)
		require.Nil(t, err)
		require.True(t, containsPeer(peers, c.peer))

		scrape = p.ScrapeSwarm(c.ih, c.peer.IPPort.IP())
		require.Equal(t, uint32(2), scrape.Incomplete)
		require.Equal(t, uint32(0), scrape.Complete)

		err = p.DeleteLeecher(c.ih, c.peer)
		require.Nil(t, err)

		peers, err = p.AnnouncePeers(c.ih, true, 50, peer)
		require.Nil(t, err)
		require.False(t, containsPeer(peers, c.peer))

		// Test PutSeeder -> Announce -> DeleteSeeder -> Announce

		err = p.PutSeeder(c.ih, c.peer)
		require.Nil(t, err)

		// Should be leecher to see the seeder
		peers, err = p.AnnouncePeers(c.ih, false, 50, peer)
		require.Nil(t, err)
		require.True(t, containsPeer(peers, c.peer))

		scrape = p.ScrapeSwarm(c.ih, c.peer.IPPort.IP())
		require.Equal(t, uint32(1), scrape.Incomplete)
		require.Equal(t, uint32(1), scrape.Complete)

		err = p.DeleteSeeder(c.ih, c.peer)
		require.Nil(t, err)

		peers, err = p.AnnouncePeers(c.ih, false, 50, peer)
		require.Nil(t, err)
		require.False(t, containsPeer(peers, c.peer))

		// Test PutLeecher -> Graduate -> Announce -> DeleteLeecher -> Announce

		err = p.PutLeecher(c.ih, c.peer)
		require.Nil(t, err)

		err = p.GraduateLeecher(c.ih, c.peer)
		require.Nil(t, err)

		// Has to be leecher to see the graduated seeder
		peers, err = p.AnnouncePeers(c.ih, false, 50, peer)
		require.Nil(t, err)
		require.True(t, containsPeer(peers, c.peer))

		// Deleting the Peer as a Leecher should have no effect
		err = p.DeleteLeecher(c.ih, c.peer)
		require.Equal(t, ErrResourceDoesNotExist, err)

		// Verify it's still there
		peers, err = p.AnnouncePeers(c.ih, false, 50, peer)
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
