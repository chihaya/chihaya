package storage

import (
	"math/rand"
	"net"
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/jzelinskie/trakr/bittorrent"
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
			IP:   net.IP(ip),
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

func Put(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, true, nil, func(i int, ps PeerStore, bd *benchData) error {
		return ps.PutSeeder(bd.infohashes[0], bd.peers[0])
	})
}

func Put1k(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, true, nil, func(i int, ps PeerStore, bd *benchData) error {
		return ps.PutSeeder(bd.infohashes[0], bd.peers[i%1000])
	})
}

func Put1kInfohash(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, true, nil, func(i int, ps PeerStore, bd *benchData) error {
		return ps.PutSeeder(bd.infohashes[i%1000], bd.peers[0])
	})
}

func Put1kInfohash1k(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, true, nil, func(i int, ps PeerStore, bd *benchData) error {
		err := ps.PutSeeder(bd.infohashes[i%1000], bd.peers[(i*3)%1000])
		return err
	})
}

func PutDelete(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, false, nil, func(i int, ps PeerStore, bd *benchData) error {
		err := ps.PutSeeder(bd.infohashes[0], bd.peers[0])
		if err != nil {
			return err
		}
		return ps.DeleteSeeder(bd.infohashes[0], bd.peers[0])
	})
}

func PutDelete1k(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, false, nil, func(i int, ps PeerStore, bd *benchData) error {
		err := ps.PutSeeder(bd.infohashes[0], bd.peers[i%1000])
		if err != nil {
			return err
		}
		return ps.DeleteSeeder(bd.infohashes[0], bd.peers[i%1000])
	})
}

func PutDelete1kInfohash(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, false, nil, func(i int, ps PeerStore, bd *benchData) error {
		err := ps.PutSeeder(bd.infohashes[i%1000], bd.peers[0])
		if err != nil {
		}
		return ps.DeleteSeeder(bd.infohashes[i%1000], bd.peers[0])
	})
}

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

func DeleteNonexist(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, true, nil, func(i int, ps PeerStore, bd *benchData) error {
		ps.DeleteSeeder(bd.infohashes[0], bd.peers[0])
		return nil
	})
}

func DeleteNonexist1k(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, true, nil, func(i int, ps PeerStore, bd *benchData) error {
		ps.DeleteSeeder(bd.infohashes[0], bd.peers[i%1000])
		return nil
	})
}

func DeleteNonexist1kInfohash(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, true, nil, func(i int, ps PeerStore, bd *benchData) error {
		ps.DeleteSeeder(bd.infohashes[i%1000], bd.peers[0])
		return nil
	})
}

func DeleteNonexist1kInfohash1k(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, true, nil, func(i int, ps PeerStore, bd *benchData) error {
		ps.DeleteSeeder(bd.infohashes[i%1000], bd.peers[(i*3)%1000])
		return nil
	})
}

func GradNonexist(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, true, nil, func(i int, ps PeerStore, bd *benchData) error {
		ps.GraduateLeecher(bd.infohashes[0], bd.peers[0])
		return nil
	})
}

func GradNonexist1k(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, true, nil, func(i int, ps PeerStore, bd *benchData) error {
		ps.GraduateLeecher(bd.infohashes[0], bd.peers[i%1000])
		return nil
	})
}

func GradNonexist1kInfohash(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, true, nil, func(i int, ps PeerStore, bd *benchData) error {
		ps.GraduateLeecher(bd.infohashes[i%1000], bd.peers[0])
		return nil
	})
}

func GradNonexist1kInfohash1k(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, true, nil, func(i int, ps PeerStore, bd *benchData) error {
		ps.GraduateLeecher(bd.infohashes[i%1000], bd.peers[(i*3)%1000])
		return nil
	})
}

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

func AnnounceLeecher(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, true, putPeers, func(i int, ps PeerStore, bd *benchData) error {
		_, err := ps.AnnouncePeers(bd.infohashes[0], false, 50, bd.peers[0])
		return err
	})
}

func AnnounceLeecher1kInfohash(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, true, putPeers, func(i int, ps PeerStore, bd *benchData) error {
		_, err := ps.AnnouncePeers(bd.infohashes[i%1000], false, 50, bd.peers[0])
		return err
	})
}

func AnnounceSeeder(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, true, putPeers, func(i int, ps PeerStore, bd *benchData) error {
		_, err := ps.AnnouncePeers(bd.infohashes[0], true, 50, bd.peers[0])
		return err
	})
}

func AnnounceSeeder1kInfohash(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, true, putPeers, func(i int, ps PeerStore, bd *benchData) error {
		_, err := ps.AnnouncePeers(bd.infohashes[i%1000], true, 50, bd.peers[0])
		return err
	})
}
