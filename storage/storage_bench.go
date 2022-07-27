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

type (
	executionFunc func(int, PeerStore, *benchData) error
	setupFunc     func(PeerStore, *benchData) error
)

func runBenchmark(b *testing.B, ps ClearablePeerStore, parallel bool, sf setupFunc, ef executionFunc) {
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

	err := ps.Clear()
	if err != nil {
		b.Fatal(err)
	}

	errChan := ps.Stop()
	for err := range errChan {
		b.Fatal(err)
	}
}

// Nop executes a no-op for each iteration.
// It should produce the same results for each PeerStore.
// This can be used to get an estimate of the impact of the benchmark harness
// on benchmark results and an estimate of the general performance of the system
// benchmarked on.
//
// Nop creates 1000 infohashes with 10 peers each.
// This is not measured in the actual benchmark.
//
// Nop can run in parallel.
func Nop(b *testing.B, ps ClearablePeerStore) {
	runBenchmark(b, ps, true, putPeers(10), func(i int, ps PeerStore, bd *benchData) error {
		return nil
	})
}

// Put benchmarks the PutSeeder method of a PeerStore by repeatedly Putting the
// same Peer for the same InfoHash.
//
// Put can run in parallel.
func Put(b *testing.B, ps ClearablePeerStore) {
	runBenchmark(b, ps, true, nil, func(i int, ps PeerStore, bd *benchData) error {
		return ps.PutSeeder(bd.infohashes[0], bd.peers[0])
	})
}

// PutSpreadPeer benchmarks the PutSeeder method of a PeerStore by cycling through 1000
// Peers and Putting them into the swarm of one infohash.
//
// PutSpreadPeer can run in parallel.
func PutSpreadPeer(b *testing.B, ps ClearablePeerStore) {
	runBenchmark(b, ps, true, nil, func(i int, ps PeerStore, bd *benchData) error {
		return ps.PutSeeder(bd.infohashes[0], bd.peers[i%1000])
	})
}

// PutSpreadInfohash benchmarks the PutSeeder method of a PeerStore by cycling
// through 1000 infohashes and putting the same peer into their swarms.
//
// PutSpreadInfohash can run in parallel.
func PutSpreadInfohash(b *testing.B, ps ClearablePeerStore) {
	runBenchmark(b, ps, true, nil, func(i int, ps PeerStore, bd *benchData) error {
		return ps.PutSeeder(bd.infohashes[i%1000], bd.peers[0])
	})
}

// PutSpreadInfohashSpreadPeer benchmarks the PutSeeder method of a PeerStore by cycling
// through 1000 infohashes and 1000 Peers and calling Put with them.
//
// PutSpreadInfohashSpreadPeer can run in parallel.
func PutSpreadInfohashSpreadPeer(b *testing.B, ps ClearablePeerStore) {
	runBenchmark(b, ps, true, nil, func(i int, ps PeerStore, bd *benchData) error {
		err := ps.PutSeeder(bd.infohashes[i%1000], bd.peers[(i*3)%1000])
		return err
	})
}

// PutDelete benchmarks the PutSeeder and DeleteSeeder methods of a PeerStore by
// calling PutSeeder followed by DeleteSeeder for one Peer and one infohash.
//
// PutDelete can not run in parallel.
func PutDelete(b *testing.B, ps ClearablePeerStore) {
	runBenchmark(b, ps, false, nil, func(i int, ps PeerStore, bd *benchData) error {
		err := ps.PutSeeder(bd.infohashes[0], bd.peers[0])
		if err != nil {
			return err
		}
		return ps.DeleteSeeder(bd.infohashes[0], bd.peers[0])
	})
}

// PutDeleteSpreadPeer benchmarks the PutSeeder and DeleteSeeder methods in the same way
// PutDelete does, but with one from 1000 Peers per iteration.
//
// PutDeleteSpreadPeer can not run in parallel.
func PutDeleteSpreadPeer(b *testing.B, ps ClearablePeerStore) {
	runBenchmark(b, ps, false, nil, func(i int, ps PeerStore, bd *benchData) error {
		err := ps.PutSeeder(bd.infohashes[0], bd.peers[i%1000])
		if err != nil {
			return err
		}
		return ps.DeleteSeeder(bd.infohashes[0], bd.peers[i%1000])
	})
}

// PutDeleteSpreadInfohash behaves like PutDeleteSpreadPeer with 1000 infohashes instead of
// 1000 Peers.
//
// PutDeleteSpreadInfohash can not run in parallel.
func PutDeleteSpreadInfohash(b *testing.B, ps ClearablePeerStore) {
	runBenchmark(b, ps, false, nil, func(i int, ps PeerStore, bd *benchData) error {
		err := ps.PutSeeder(bd.infohashes[i%1000], bd.peers[0])
		if err != nil {
			return err
		}
		return ps.DeleteSeeder(bd.infohashes[i%1000], bd.peers[0])
	})
}

// PutDeleteSpreadInfohashSpreadPeer behaves like PutDeleteSpreadPeer with 1000 infohashes in
// addition to 1000 Peers.
//
// PutDeleteSpreadInfohashSpreadPeer can not run in parallel.
func PutDeleteSpreadInfohashSpreadPeer(b *testing.B, ps ClearablePeerStore) {
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
func DeleteNonexist(b *testing.B, ps ClearablePeerStore) {
	runBenchmark(b, ps, true, nil, func(i int, ps PeerStore, bd *benchData) error {
		_ = ps.DeleteSeeder(bd.infohashes[0], bd.peers[0])
		return nil
	})
}

// DeleteNonexistSpreadPeer benchmarks the DeleteSeeder method of a PeerStore by
// attempting to delete one of 1000 nonexistent Peers.
//
// DeleteNonexist can run in parallel.
func DeleteNonexistSpreadPeer(b *testing.B, ps ClearablePeerStore) {
	runBenchmark(b, ps, true, nil, func(i int, ps PeerStore, bd *benchData) error {
		_ = ps.DeleteSeeder(bd.infohashes[0], bd.peers[i%1000])
		return nil
	})
}

// DeleteNonexistSpreadInfohash benchmarks the DeleteSeeder method of a PeerStore by
// attempting to delete one Peer from one of 1000 infohashes.
//
// DeleteNonexistSpreadInfohash can run in parallel.
func DeleteNonexistSpreadInfohash(b *testing.B, ps ClearablePeerStore) {
	runBenchmark(b, ps, true, nil, func(i int, ps PeerStore, bd *benchData) error {
		_ = ps.DeleteSeeder(bd.infohashes[i%1000], bd.peers[0])
		return nil
	})
}

// DeleteNonexistSpreadInfohashSpreadPeer benchmarks the Delete method of a PeerStore by
// attempting to delete one of 1000 Peers from one of 1000 Infohashes.
//
// DeleteNonexistSpreadInfohashSpreadPeer can run in parallel.
func DeleteNonexistSpreadInfohashSpreadPeer(b *testing.B, ps ClearablePeerStore) {
	runBenchmark(b, ps, true, nil, func(i int, ps PeerStore, bd *benchData) error {
		_ = ps.DeleteSeeder(bd.infohashes[i%1000], bd.peers[(i*3)%1000])
		return nil
	})
}

// GradNonexist benchmarks the GraduateLeecher method of a PeerStore by
// attempting to graduate a nonexistent Peer.
//
// GradNonexist can run in parallel.
func GradNonexist(b *testing.B, ps ClearablePeerStore) {
	runBenchmark(b, ps, true, nil, func(i int, ps PeerStore, bd *benchData) error {
		_ = ps.GraduateLeecher(bd.infohashes[0], bd.peers[0])
		return nil
	})
}

// GradNonexistSpreadPeer benchmarks the GraduateLeecher method of a PeerStore by
// attempting to graduate one of 1000 nonexistent Peers.
//
// GradNonexistSpreadPeer can run in parallel.
func GradNonexistSpreadPeer(b *testing.B, ps ClearablePeerStore) {
	runBenchmark(b, ps, true, nil, func(i int, ps PeerStore, bd *benchData) error {
		_ = ps.GraduateLeecher(bd.infohashes[0], bd.peers[i%1000])
		return nil
	})
}

// GradNonexistSpreadInfohash benchmarks the GraduateLeecher method of a PeerStore
// by attempting to graduate a nonexistent Peer for one of 100 Infohashes.
//
// GradNonexistSpreadInfohash can run in parallel.
func GradNonexistSpreadInfohash(b *testing.B, ps ClearablePeerStore) {
	runBenchmark(b, ps, true, nil, func(i int, ps PeerStore, bd *benchData) error {
		_ = ps.GraduateLeecher(bd.infohashes[i%1000], bd.peers[0])
		return nil
	})
}

// GradNonexistSpreadInfohashSpreadPeer benchmarks the GraduateLeecher method of a PeerStore
// by attempting to graduate one of 1000 nonexistent Peers for one of 1000
// infohashes.
//
// GradNonexistSpreadInfohashSpreadPeer can run in parallel.
func GradNonexistSpreadInfohashSpreadPeer(b *testing.B, ps ClearablePeerStore) {
	runBenchmark(b, ps, true, nil, func(i int, ps PeerStore, bd *benchData) error {
		_ = ps.GraduateLeecher(bd.infohashes[i%1000], bd.peers[(i*3)%1000])
		return nil
	})
}

// PutGradDelete benchmarks the PutLeecher, GraduateLeecher and DeleteSeeder
// methods of a PeerStore by adding one leecher to a swarm, promoting it to a
// seeder and deleting the seeder.
//
// PutGradDelete can not run in parallel.
func PutGradDelete(b *testing.B, ps ClearablePeerStore) {
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

// PutGradDeleteSpreadPeer behaves like PutGradDelete with one of 1000 Peers.
//
// PutGradDeleteSpreadPeer can not run in parallel.
func PutGradDeleteSpreadPeer(b *testing.B, ps ClearablePeerStore) {
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

// PutGradDeleteSpreadInfohash behaves like PutGradDelete with one of 1000
// infohashes.
//
// PutGradDeleteSpreadInfohash can not run in parallel.
func PutGradDeleteSpreadInfohash(b *testing.B, ps ClearablePeerStore) {
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

// PutGradDeleteSpreadInfohashSpreadPeer behaves like PutGradDelete with one of 1000 Peers
// and one of 1000 infohashes.
//
// PutGradDeleteSpreadInfohash can not run in parallel.
func PutGradDeleteSpreadInfohashSpreadPeer(b *testing.B, ps ClearablePeerStore) {
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

func putPeers(numPeers int) func(PeerStore, *benchData) error {
	return func(ps PeerStore, bd *benchData) error {
		for i := 0; i < 1000; i++ {
			for j := 0; j < numPeers; j++ {
				var err error
				if j < numPeers/2 {
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
}

// AnnounceLeecherLarge benchmarks the AnnouncePeers method of a PeerStore for
// announcing a leecher.
// The swarm announced to has 500 seeders and 500 leechers.
//
// AnnounceLeecherLarge can run in parallel.
func AnnounceLeecherLarge(b *testing.B, ps ClearablePeerStore) {
	runBenchmark(b, ps, true, putPeers(1000), func(i int, ps PeerStore, bd *benchData) error {
		_, err := ps.AnnouncePeers(bd.infohashes[0], false, 50, bd.peers[0])
		return err
	})
}

// AnnounceLeecherLargeSpreadInfohash behaves like AnnounceLeecherLarge with one of 1000
// infohashes.
//
// AnnounceLeecherLargeSpreadInfohash can run in parallel.
func AnnounceLeecherLargeSpreadInfohash(b *testing.B, ps ClearablePeerStore) {
	runBenchmark(b, ps, true, putPeers(1000), func(i int, ps PeerStore, bd *benchData) error {
		_, err := ps.AnnouncePeers(bd.infohashes[i%1000], false, 50, bd.peers[0])
		return err
	})
}

// AnnounceSeederLarge behaves like AnnounceLeecherLarge with a seeder instead of a
// leecher.
//
// AnnounceSeederLarge can run in parallel.
func AnnounceSeederLarge(b *testing.B, ps ClearablePeerStore) {
	runBenchmark(b, ps, true, putPeers(1000), func(i int, ps PeerStore, bd *benchData) error {
		_, err := ps.AnnouncePeers(bd.infohashes[0], true, 50, bd.peers[0])
		return err
	})
}

// AnnounceSeederLargeSpreadInfohash behaves like AnnounceSeederLarge with one of 1000
// infohashes.
//
// AnnounceSeederLargeSpreadInfohash can run in parallel.
func AnnounceSeederLargeSpreadInfohash(b *testing.B, ps ClearablePeerStore) {
	runBenchmark(b, ps, true, putPeers(1000), func(i int, ps PeerStore, bd *benchData) error {
		_, err := ps.AnnouncePeers(bd.infohashes[i%1000], true, 50, bd.peers[0])
		return err
	})
}

// AnnounceLeecherSmall benchmarks the AnnouncePeers method of a PeerStore for
// announcing a leecher.
// The swarm announced to has 5 seeders and 5 leechers.
//
// AnnounceLeecherSmall can run in parallel.
func AnnounceLeecherSmall(b *testing.B, ps ClearablePeerStore) {
	runBenchmark(b, ps, true, putPeers(10), func(i int, ps PeerStore, bd *benchData) error {
		_, err := ps.AnnouncePeers(bd.infohashes[0], false, 50, bd.peers[0])
		return err
	})
}

// AnnounceLeecherSmallSpreadInfohash behaves like AnnounceLeecherSmall with one of 1000
// infohashes.
//
// AnnounceLeecherSmallSpreadInfohash can run in parallel.
func AnnounceLeecherSmallSpreadInfohash(b *testing.B, ps ClearablePeerStore) {
	runBenchmark(b, ps, true, putPeers(10), func(i int, ps PeerStore, bd *benchData) error {
		_, err := ps.AnnouncePeers(bd.infohashes[i%1000], false, 50, bd.peers[0])
		return err
	})
}

// AnnounceSeederSmall behaves like AnnounceLeecherSmall with a seeder instead of a
// leecher.
//
// AnnounceSeederSmall can run in parallel.
func AnnounceSeederSmall(b *testing.B, ps ClearablePeerStore) {
	runBenchmark(b, ps, true, putPeers(10), func(i int, ps PeerStore, bd *benchData) error {
		_, err := ps.AnnouncePeers(bd.infohashes[0], true, 50, bd.peers[0])
		return err
	})
}

// AnnounceSeederSmallSpreadInfohash behaves like AnnounceSeederSmall with one of 1000
// infohashes.
//
// AnnounceSeederSmallSpreadInfohash can run in parallel.
func AnnounceSeederSmallSpreadInfohash(b *testing.B, ps ClearablePeerStore) {
	runBenchmark(b, ps, true, putPeers(10), func(i int, ps PeerStore, bd *benchData) error {
		_, err := ps.AnnouncePeers(bd.infohashes[i%1000], true, 50, bd.peers[0])
		return err
	})
}

// ScrapeSwarmLarge benchmarks the ScrapeSwarm method of a PeerStore.
// The swarm scraped has 500 seeders and 500 leechers.
//
// ScrapeSwarmLarge can run in parallel.
func ScrapeSwarmLarge(b *testing.B, ps ClearablePeerStore) {
	runBenchmark(b, ps, true, putPeers(1000), func(i int, ps PeerStore, bd *benchData) error {
		ps.ScrapeSwarm(bd.infohashes[0], bittorrent.IPv4)
		return nil
	})
}

// ScrapeSwarmLargeSpreadInfohash behaves like ScrapeSwarmLarge with one of 1000 infohashes.
//
// ScrapeSwarmLargeSpreadInfohash can run in parallel.
func ScrapeSwarmLargeSpreadInfohash(b *testing.B, ps ClearablePeerStore) {
	runBenchmark(b, ps, true, putPeers(1000), func(i int, ps PeerStore, bd *benchData) error {
		ps.ScrapeSwarm(bd.infohashes[i%1000], bittorrent.IPv4)
		return nil
	})
}

// ScrapeSwarmSmall benchmarks the ScrapeSwarm method of a PeerStore.
// The swarm scraped has 5 seeders and 5 leechers.
//
// ScrapeSwarmSmall can run in parallel.
func ScrapeSwarmSmall(b *testing.B, ps ClearablePeerStore) {
	runBenchmark(b, ps, true, putPeers(10), func(i int, ps PeerStore, bd *benchData) error {
		ps.ScrapeSwarm(bd.infohashes[0], bittorrent.IPv4)
		return nil
	})
}

// ScrapeSwarmSmallSpreadInfohash behaves like ScrapeSwarmSmall with one of 1000 infohashes.
//
// ScrapeSwarmSmallSpreadInfohash can run in parallel.
func ScrapeSwarmSmallSpreadInfohash(b *testing.B, ps ClearablePeerStore) {
	runBenchmark(b, ps, true, putPeers(10), func(i int, ps PeerStore, bd *benchData) error {
		ps.ScrapeSwarm(bd.infohashes[i%1000], bittorrent.IPv4)
		return nil
	})
}
