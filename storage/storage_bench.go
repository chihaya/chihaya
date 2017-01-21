package storage

import (
	"math/rand"
	"net"
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/chihaya/chihaya/bittorrent"
)

type benchData struct {
	infohashes [1000]bittorrent.InfoHash
	peers      [1000]bittorrent.Peer
}

func generateInfohashes() (a [1000]bittorrent.InfoHash) {
	r := rand.New(rand.NewSource(0))
	for i := range a {
		b := [20]byte{}
		n, err := r.Read(b[:])
		if err != nil || n != 20 {
			panic("unable to create random bytes")
		}
		a[i] = bittorrent.InfoHash(b)
	}

	return
}

func generatePeers() (a [1000]bittorrent.Peer) {
	r := rand.New(rand.NewSource(0))
	for i := range a {
		ip := make([]byte, 4)
		n, err := r.Read(ip)
		if err != nil || n != 4 {
			panic("unable to create random bytes")
		}
		id := [20]byte{}
		n, err = r.Read(id[:])
		if err != nil || n != 20 {
			panic("unable to create random bytes")
		}
		port := uint16(r.Uint32())
		a[i] = bittorrent.Peer{
			ID:   bittorrent.PeerID(id),
			IP:   bittorrent.IP{IP: net.IP(ip), AddressFamily: bittorrent.IPv4},
			Port: port,
		}
	}

	return
}

type executionFunc func(int, PeerStore, *benchData) error
type setupFunc func(PeerStore, *benchData) error

func runBenchmark(b *testing.B, ps PeerStore, parallel bool, sf setupFunc, ef executionFunc) {
	bd := &benchData{generateInfohashes(), generatePeers()}
	spacing := int32(1000 / runtime.NumCPU())
	if sf != nil {
		err := sf(ps, bd)
		if err != nil {
			b.Fatal(err)
		}
	}
	offset := int32(0)

	b.ResetTimer()
	if parallel {
		b.RunParallel(func(pb *testing.PB) {
			i := int(atomic.AddInt32(&offset, spacing))
			for pb.Next() {
				err := ef(i, ps, bd)
				if err != nil {
					b.Fatal(err)
				}
				i++
			}
		})
	} else {
		for i := 0; i < b.N; i++ {
			err := ef(i, ps, bd)
			if err != nil {
				b.Fatal(err)
			}
		}
	}
	b.StopTimer()

	errChan := ps.Stop()
	for err := range errChan {
		b.Fatal(err)
	}
}

// Put benchmarks the PutSeeder method of a PeerStore by repeatedly Putting the
// same Peer for the same InfoHash.
//
// Put can run in parallel.
func Put(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, true, nil, func(i int, ps PeerStore, bd *benchData) error {
		return ps.PutSeeder(bd.infohashes[0], bd.peers[0])
	})
}

// Put1k benchmarks the PutSeeder method of a PeerStore by cycling through 1000
// Peers and Putting them into the swarm of one infohash.
//
// Put1k can run in parallel.
func Put1k(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, true, nil, func(i int, ps PeerStore, bd *benchData) error {
		return ps.PutSeeder(bd.infohashes[0], bd.peers[i%1000])
	})
}

// Put1kInfohash benchmarks the PutSeeder method of a PeerStore by cycling
// through 1000 infohashes and putting the same peer into their swarms.
//
// Put1kInfohash can run in parallel.
func Put1kInfohash(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, true, nil, func(i int, ps PeerStore, bd *benchData) error {
		return ps.PutSeeder(bd.infohashes[i%1000], bd.peers[0])
	})
}

// Put1kInfohash1k benchmarks the PutSeeder method of a PeerStore by cycling
// through 1000 infohashes and 1000 Peers and calling Put with them.
//
// Put1kInfohash1k can run in parallel.
func Put1kInfohash1k(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, true, nil, func(i int, ps PeerStore, bd *benchData) error {
		err := ps.PutSeeder(bd.infohashes[i%1000], bd.peers[(i*3)%1000])
		return err
	})
}

// PutDelete benchmarks the PutSeeder and DeleteSeeder methods of a PeerStore by
// calling PutSeeder followed by DeleteSeeder for one Peer and one infohash.
//
// PutDelete can not run in parallel.
func PutDelete(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, false, nil, func(i int, ps PeerStore, bd *benchData) error {
		err := ps.PutSeeder(bd.infohashes[0], bd.peers[0])
		if err != nil {
			return err
		}
		return ps.DeleteSeeder(bd.infohashes[0], bd.peers[0])
	})
}

// PutDelete1k benchmarks the PutSeeder and DeleteSeeder methods in the same way
// PutDelete does, but with one from 1000 Peers per iteration.
//
// PutDelete1k can not run in parallel.
func PutDelete1k(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, false, nil, func(i int, ps PeerStore, bd *benchData) error {
		err := ps.PutSeeder(bd.infohashes[0], bd.peers[i%1000])
		if err != nil {
			return err
		}
		return ps.DeleteSeeder(bd.infohashes[0], bd.peers[i%1000])
	})
}

// PutDelete1kInfohash behaves like PutDelete1k with 1000 infohashes instead of
// 1000 Peers.
//
// PutDelete1kInfohash can not run in parallel.
func PutDelete1kInfohash(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, false, nil, func(i int, ps PeerStore, bd *benchData) error {
		err := ps.PutSeeder(bd.infohashes[i%1000], bd.peers[0])
		if err != nil {
		}
		return ps.DeleteSeeder(bd.infohashes[i%1000], bd.peers[0])
	})
}

// PutDelete1kInfohash1k behaves like PutDelete1k with 1000 infohashes in
// addition to 1000 Peers.
//
// PutDelete1kInfohash1k can not run in parallel.
func PutDelete1kInfohash1k(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, false, nil, func(i int, ps PeerStore, bd *benchData) error {
		err := ps.PutSeeder(bd.infohashes[i%1000], bd.peers[(i*3)%1000])
		if err != nil {
			return err
		}
		err = ps.DeleteSeeder(bd.infohashes[i%1000], bd.peers[(i*3)%1000])
		return err
	})
}

// DeleteNonexist benchmarks the DeleteSeeder method of a PeerStore by
// attempting to delete a Peer that is nonexistent.
//
// DeleteNonexist can run in parallel.
func DeleteNonexist(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, true, nil, func(i int, ps PeerStore, bd *benchData) error {
		ps.DeleteSeeder(bd.infohashes[0], bd.peers[0])
		return nil
	})
}

// DeleteNonexist1k benchmarks the DeleteSeeder method of a PeerStore by
// attempting to delete one of 1000 nonexistent Peers.
//
// DeleteNonexist can run in parallel.
func DeleteNonexist1k(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, true, nil, func(i int, ps PeerStore, bd *benchData) error {
		ps.DeleteSeeder(bd.infohashes[0], bd.peers[i%1000])
		return nil
	})
}

// DeleteNonexist1kInfohash benchmarks the DeleteSeeder method of a PeerStore by
// attempting to delete one Peer from one of 1000 infohashes.
//
// DeleteNonexist1kInfohash can run in parallel.
func DeleteNonexist1kInfohash(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, true, nil, func(i int, ps PeerStore, bd *benchData) error {
		ps.DeleteSeeder(bd.infohashes[i%1000], bd.peers[0])
		return nil
	})
}

// DeleteNonexist1kInfohash1k benchmarks the Delete method of a PeerStore by
// attempting to delete one of 1000 Peers from one of 1000 Infohashes.
//
// DeleteNonexist1kInfohash1k can run in parallel.
func DeleteNonexist1kInfohash1k(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, true, nil, func(i int, ps PeerStore, bd *benchData) error {
		ps.DeleteSeeder(bd.infohashes[i%1000], bd.peers[(i*3)%1000])
		return nil
	})
}

// GradNonexist benchmarks the GraduateLeecher method of a PeerStore by
// attempting to graduate a nonexistent Peer.
//
// GradNonexist can run in parallel.
func GradNonexist(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, true, nil, func(i int, ps PeerStore, bd *benchData) error {
		ps.GraduateLeecher(bd.infohashes[0], bd.peers[0])
		return nil
	})
}

// GradNonexist1k benchmarks the GraduateLeecher method of a PeerStore by
// attempting to graduate one of 1000 nonexistent Peers.
//
// GradNonexist1k can run in parallel.
func GradNonexist1k(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, true, nil, func(i int, ps PeerStore, bd *benchData) error {
		ps.GraduateLeecher(bd.infohashes[0], bd.peers[i%1000])
		return nil
	})
}

// GradNonexist1kInfohash benchmarks the GraduateLeecher method of a PeerStore
// by attempting to graduate a nonexistent Peer for one of 100 Infohashes.
//
// GradNonexist1kInfohash can run in parallel.
func GradNonexist1kInfohash(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, true, nil, func(i int, ps PeerStore, bd *benchData) error {
		ps.GraduateLeecher(bd.infohashes[i%1000], bd.peers[0])
		return nil
	})
}

// GradNonexist1kInfohash1k benchmarks the GraduateLeecher method of a PeerStore
// by attempting to graduate one of 1000 nonexistent Peers for one of 1000
// infohashes.
//
// GradNonexist1kInfohash1k can run in parallel.
func GradNonexist1kInfohash1k(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, true, nil, func(i int, ps PeerStore, bd *benchData) error {
		ps.GraduateLeecher(bd.infohashes[i%1000], bd.peers[(i*3)%1000])
		return nil
	})
}

// PutGradDelete benchmarks the PutLeecher, GraduateLeecher and DeleteSeeder
// methods of a PeerStore by adding one leecher to a swarm, promoting it to a
// seeder and deleting the seeder.
//
// PutGradDelete can not run in parallel.
func PutGradDelete(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, false, nil, func(i int, ps PeerStore, bd *benchData) error {
		err := ps.PutLeecher(bd.infohashes[0], bd.peers[0])
		if err != nil {
			return err
		}
		err = ps.GraduateLeecher(bd.infohashes[0], bd.peers[0])
		if err != nil {
			return err
		}
		return ps.DeleteSeeder(bd.infohashes[0], bd.peers[0])
	})
}

// PutGradDelete1k behaves like PutGradDelete with one of 1000 Peers.
//
// PutGradDelete1k can not run in parallel.
func PutGradDelete1k(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, false, nil, func(i int, ps PeerStore, bd *benchData) error {
		err := ps.PutLeecher(bd.infohashes[0], bd.peers[i%1000])
		if err != nil {
			return err
		}
		err = ps.GraduateLeecher(bd.infohashes[0], bd.peers[i%1000])
		if err != nil {
			return err
		}
		return ps.DeleteSeeder(bd.infohashes[0], bd.peers[i%1000])
	})
}

// PutGradDelete1kInfohash behaves like PutGradDelete with one of 1000
// infohashes.
//
// PutGradDelete1kInfohash can not run in parallel.
func PutGradDelete1kInfohash(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, false, nil, func(i int, ps PeerStore, bd *benchData) error {
		err := ps.PutLeecher(bd.infohashes[i%1000], bd.peers[0])
		if err != nil {
			return err
		}
		err = ps.GraduateLeecher(bd.infohashes[i%1000], bd.peers[0])
		if err != nil {
			return err
		}
		return ps.DeleteSeeder(bd.infohashes[i%1000], bd.peers[0])
	})
}

// PutGradDelete1kInfohash1k behaves like PutGradDelete with one of 1000 Peers
// and one of 1000 infohashes.
//
// PutGradDelete1kInfohash can not run in parallel.
func PutGradDelete1kInfohash1k(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, false, nil, func(i int, ps PeerStore, bd *benchData) error {
		err := ps.PutLeecher(bd.infohashes[i%1000], bd.peers[(i*3)%1000])
		if err != nil {
			return err
		}
		err = ps.GraduateLeecher(bd.infohashes[i%1000], bd.peers[(i*3)%1000])
		if err != nil {
			return err
		}
		err = ps.DeleteSeeder(bd.infohashes[i%1000], bd.peers[(i*3)%1000])
		return err
	})
}

func putPeers(ps PeerStore, bd *benchData) error {
	for i := 0; i < 1000; i++ {
		for j := 0; j < 1000; j++ {
			var err error
			if j < 1000/2 {
				err = ps.PutLeecher(bd.infohashes[i], bd.peers[j])
			} else {
				err = ps.PutSeeder(bd.infohashes[i], bd.peers[j])
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// AnnounceLeecher benchmarks the AnnouncePeers method of a PeerStore for
// announcing a leecher.
// The swarm announced to has 500 seeders and 500 leechers.
//
// AnnounceLeecher can run in parallel.
func AnnounceLeecher(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, true, putPeers, func(i int, ps PeerStore, bd *benchData) error {
		_, err := ps.AnnouncePeers(bd.infohashes[0], false, 50, bd.peers[0])
		return err
	})
}

// AnnounceLeecher1kInfohash behaves like AnnounceLeecher with one of 1000
// infohashes.
//
// AnnounceLeecher1kInfohash can run in parallel.
func AnnounceLeecher1kInfohash(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, true, putPeers, func(i int, ps PeerStore, bd *benchData) error {
		_, err := ps.AnnouncePeers(bd.infohashes[i%1000], false, 50, bd.peers[0])
		return err
	})
}

// AnnounceSeeder behaves like AnnounceLeecher with a seeder instead of a
// leecher.
//
// AnnounceSeeder can run in parallel.
func AnnounceSeeder(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, true, putPeers, func(i int, ps PeerStore, bd *benchData) error {
		_, err := ps.AnnouncePeers(bd.infohashes[0], true, 50, bd.peers[0])
		return err
	})
}

// AnnounceSeeder1kInfohash behaves like AnnounceSeeder with one of 1000
// infohashes.
//
// AnnounceSeeder1kInfohash can run in parallel.
func AnnounceSeeder1kInfohash(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, true, putPeers, func(i int, ps PeerStore, bd *benchData) error {
		_, err := ps.AnnouncePeers(bd.infohashes[i%1000], true, 50, bd.peers[0])
		return err
	})
}
