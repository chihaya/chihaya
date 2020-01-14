package storage

import (
	"bytes"
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

		_, err = p.AnnouncePeers(c.ih, false, 50, peer)
		require.Equal(t, ErrResourceDoesNotExist, err)

		// Test empty scrapes response for non-existent swarms.
		scrapes := p.ScrapeSwarms([]bittorrent.InfoHash{c.ih}, c.peer.IP.AddressFamily)
		require.Equal(t, 1, len(scrapes))
		require.Equal(t, uint32(0), scrapes[0].Complete)
		require.Equal(t, uint32(0), scrapes[0].Incomplete)
		require.Equal(t, uint32(0), scrapes[0].Snatches)

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

		scrapes = p.ScrapeSwarms([]bittorrent.InfoHash{c.ih}, c.peer.IP.AddressFamily)
		require.Equal(t, 1, len(scrapes))
		require.Equal(t, uint32(2), scrapes[0].Incomplete)
		require.Equal(t, uint32(0), scrapes[0].Complete)

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

		scrapes = p.ScrapeSwarms([]bittorrent.InfoHash{c.ih}, c.peer.IP.AddressFamily)
		require.Equal(t, 1, len(scrapes))
		require.Equal(t, uint32(1), scrapes[0].Incomplete)
		require.Equal(t, uint32(1), scrapes[0].Complete)

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

// TestFullscrape tests whether a storage implementation correctly implements
// fullscrapes.
func TestFullscrape(t *testing.T, ps PeerStore) {
	testData := []struct {
		ih           bittorrent.InfoHash
		seeders      []bittorrent.Peer
		leechers     []bittorrent.Peer
		v4Complete   uint32
		v6Complete   uint32
		v4Incomplete uint32
		v6Incomplete uint32
	}{
		{
			bittorrent.InfoHashFromString("00000000000000000001"),
			[]bittorrent.Peer{
				{
					ID:   bittorrent.PeerIDFromString("00000000000000000001"),
					Port: 1,
					IP: bittorrent.IP{
						IP:            net.ParseIP("1.1.1.1").To4(),
						AddressFamily: bittorrent.IPv4}},
				{
					ID:   bittorrent.PeerIDFromString("00000000000000000002"),
					Port: 2,
					IP: bittorrent.IP{
						IP:            net.ParseIP("1.1.1.2").To4(),
						AddressFamily: bittorrent.IPv4}},
			},
			[]bittorrent.Peer{
				{
					ID:   bittorrent.PeerIDFromString("00000000000000000003"),
					Port: 3,
					IP: bittorrent.IP{
						IP:            net.ParseIP("1.1.1.3").To4(),
						AddressFamily: bittorrent.IPv4}},
			},
			2,
			0,
			1,
			0,
		},
		{
			bittorrent.InfoHashFromString("00000000000000000002"),
			[]bittorrent.Peer{
				{
					ID:   bittorrent.PeerIDFromString("00000000000000000001"),
					Port: 1,
					IP: bittorrent.IP{
						IP:            net.ParseIP("1.1.1.1").To4(),
						AddressFamily: bittorrent.IPv4}},
				{
					ID:   bittorrent.PeerIDFromString("00000000000000000002"),
					Port: 2,
					IP: bittorrent.IP{
						IP:            net.ParseIP("abab::0001"),
						AddressFamily: bittorrent.IPv6}},
			},
			[]bittorrent.Peer{
				{
					ID:   bittorrent.PeerIDFromString("00000000000000000003"),
					Port: 3,
					IP: bittorrent.IP{
						IP:            net.ParseIP("abab::0003"),
						AddressFamily: bittorrent.IPv6}},
			},
			1,
			1,
			0,
			1,
		},
	}

	for _, td := range testData {
		for _, seeder := range td.seeders {
			err := ps.PutSeeder(td.ih, seeder)
			require.Nil(t, err)
		}
		for _, leecher := range td.leechers {
			err := ps.PutLeecher(td.ih, leecher)
			require.Nil(t, err)
		}
	}

	v4Full := ps.ScrapeSwarms(nil, bittorrent.IPv4)
	require.Len(t, v4Full, 2)
	v6Full := ps.ScrapeSwarms(nil, bittorrent.IPv6)
	require.Len(t, v6Full, 1)

	for _, scrape := range v4Full {
		for _, td := range testData {
			if bytes.Equal(td.ih[:], scrape.InfoHash[:]) {
				require.Equal(t, td.v4Complete, scrape.Complete)
				require.Equal(t, td.v4Incomplete, scrape.Incomplete)
				break
			}
		}
	}

	for _, scrape := range v6Full {
		for _, td := range testData {
			if bytes.Equal(td.ih[:], scrape.InfoHash[:]) {
				require.Equal(t, td.v6Complete, scrape.Complete)
				require.Equal(t, td.v6Incomplete, scrape.Incomplete)
				break
			}
		}
	}

	// Clean up
	for _, td := range testData {
		for _, seeder := range td.seeders {
			err := ps.DeleteSeeder(td.ih, seeder)
			require.Nil(t, err)
		}
		for _, leecher := range td.leechers {
			err := ps.DeleteLeecher(td.ih, leecher)
			require.Nil(t, err)
		}
	}

	e := ps.Stop()
	require.Nil(t, <-e)
}
