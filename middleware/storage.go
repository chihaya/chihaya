package middleware

import (
	"time"

	"github.com/jzelinskie/trakr/bittorrent"
	"github.com/jzelinskie/trakr/stopper"
)

// ErrResourceDoesNotExist is the error returned by all delete methods in the
// store if the requested resource does not exist.
var ErrResourceDoesNotExist = bittorrent.ClientError("resource does not exist")

// PeerStore is an interface that abstracts the interactions of storing and
// manipulating Peers such that it can be implemented for various data stores.
type PeerStore interface {
	// PutSeeder adds a Seeder to the Swarm identified by the provided infoHash.
	PutSeeder(infoHash bittorrent.InfoHash, p bittorrent.Peer) error

	// DeleteSeeder removes a Seeder from the Swarm identified by the provided
	// infoHash.
	//
	// If the Swarm or Peer does not exist, this function should return
	// ErrResourceDoesNotExist.
	DeleteSeeder(infoHash bittorrent.InfoHash, p bittorrent.Peer) error

	// PutLeecher adds a Leecher to the Swarm identified by the provided
	// infoHash.
	PutLeecher(infoHash bittorrent.InfoHash, p bittorrent.Peer) error

	// DeleteLeecher removes a Leecher from the Swarm identified by the provided
	// infoHash.
	//
	// If the Swarm or Peer does not exist, this function should return
	// ErrResourceDoesNotExist.
	DeleteLeecher(infoHash bittorrent.InfoHash, p bittorrent.Peer) error

	// GraduateLeecher promotes a Leecher to a Seeder in the Swarm identified by
	// the provided infoHash.
	//
	// If the given Peer is not present as a Leecher, add the Peer as a Seeder
	// and return no error.
	GraduateLeecher(infoHash bittorrent.InfoHash, p bittorrent.Peer) error

	// AnnouncePeers is a best effort attempt to return Peers from the Swarm
	// identified by the provided infoHash. The returned Peers are required to be
	// either all IPv4 or all IPv6.
	//
	// The returned Peers should strive be:
	// - as close to length equal to numWant as possible without going over
	// - all IPv4 or all IPv6 depending on the provided peer
	// - if seeder is true, should ideally return more leechers than seeders
	// - if seeder is false, should ideally return more seeders than leechers
	AnnouncePeers(infoHash bittorrent.InfoHash, seeder bool, numWant int, p bittorrent.Peer) (peers []bittorrent.Peer, err error)

	// CollectGarbage deletes all Peers from the PeerStore which are older than
	// the cutoff time. This function must be able to execute while other methods
	// on this interface are being executed in parallel.
	CollectGarbage(cutoff time.Time) error

	// Stopper is an interface that expects a Stop method to stops the PeerStore.
	// For more details see the documentation in the stopper package.
	stopper.Stopper
}
