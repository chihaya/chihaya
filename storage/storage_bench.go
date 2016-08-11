package storage

import (
	"fmt"
	"net"
	"testing"

	"github.com/jzelinskie/trakr/bittorrent"
)

type benchData struct {
	infohashes [1000]bittorrent.InfoHash
	peers      [1000]bittorrent.Peer
}

func generateInfohashes() (a [1000]bittorrent.InfoHash) {
	b := make([]byte, 2)
	for i := range a {
		b[0] = byte(i)
		b[1] = byte(i >> 8)
		a[i] = bittorrent.InfoHash([20]byte{b[0], b[1]})
	}

	return
}

func generatePeers() (a [1000]bittorrent.Peer) {
	b := make([]byte, 2)
	for i := range a {
		b[0] = byte(i)
		b[1] = byte(i >> 8)
		a[i] = bittorrent.Peer{
			ID:   bittorrent.PeerID([20]byte{b[0], b[1]}),
			IP:   net.ParseIP(fmt.Sprintf("64.%d.%d.64", b[0], b[1])),
			Port: uint16(i),
		}
	}

	return
}

type executionFunc func(int, PeerStore, *benchData) error
type setupFunc func(PeerStore, *benchData) error

func runBenchmark(b *testing.B, ps PeerStore, sf setupFunc, ef executionFunc) {
	bd := &benchData{generateInfohashes(), generatePeers()}
	if sf != nil {
		err := sf(ps, bd)
		if err != nil {
			b.Fatal(err)
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := ef(i, ps, bd)
		if err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
}

func Put(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, nil, func(i int, ps PeerStore, bd *benchData) error {
		return ps.PutSeeder(bd.infohashes[0], bd.peers[0])
	})
}

func Put1k(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, nil, func(i int, ps PeerStore, bd *benchData) error {
		return ps.PutSeeder(bd.infohashes[0], bd.peers[i%1000])
	})
}

func Put1kInfohash(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, nil, func(i int, ps PeerStore, bd *benchData) error {
		return ps.PutSeeder(bd.infohashes[i%1000], bd.peers[0])
	})
}

func Put1kInfohash1k(b *testing.B, ps PeerStore) {
	j := 0
	runBenchmark(b, ps, nil, func(i int, ps PeerStore, bd *benchData) error {
		err := ps.PutSeeder(bd.infohashes[i%1000], bd.peers[j%1000])
		j += 3
		return err
	})
}

func PutDelete(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, nil, func(i int, ps PeerStore, bd *benchData) error {
		err := ps.PutSeeder(bd.infohashes[0], bd.peers[0])
		if err != nil {
			return err
		}
		return ps.DeleteSeeder(bd.infohashes[0], bd.peers[0])
	})
}

func PutDelete1k(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, nil, func(i int, ps PeerStore, bd *benchData) error {
		err := ps.PutSeeder(bd.infohashes[0], bd.peers[i%1000])
		if err != nil {
			return err
		}
		return ps.DeleteSeeder(bd.infohashes[0], bd.peers[i%1000])
	})
}

func PutDelete1kInfohash(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, nil, func(i int, ps PeerStore, bd *benchData) error {
		err := ps.PutSeeder(bd.infohashes[i%1000], bd.peers[0])
		if err != nil {
		}
		return ps.DeleteSeeder(bd.infohashes[i%1000], bd.peers[0])
	})
}

func PutDelete1kInfohash1k(b *testing.B, ps PeerStore) {
	j := 0
	runBenchmark(b, ps, nil, func(i int, ps PeerStore, bd *benchData) error {
		err := ps.PutSeeder(bd.infohashes[i%1000], bd.peers[j%1000])
		if err != nil {
			return err
		}
		err = ps.DeleteSeeder(bd.infohashes[i%1000], bd.peers[j%1000])
		j += 3
		return err
	})
}

func DeleteNonexist(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, nil, func(i int, ps PeerStore, bd *benchData) error {
		return ps.DeleteSeeder(bd.infohashes[0], bd.peers[0])
	})
}

func DeleteNonexist1k(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, nil, func(i int, ps PeerStore, bd *benchData) error {
		return ps.DeleteSeeder(bd.infohashes[0], bd.peers[i%1000])
	})
}

func DeleteNonexist1kInfohash(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, nil, func(i int, ps PeerStore, bd *benchData) error {
		return ps.DeleteSeeder(bd.infohashes[i%1000], bd.peers[0])
	})
}

func DeleteNonexist1kInfohash1k(b *testing.B, ps PeerStore) {
	j := 0
	runBenchmark(b, ps, nil, func(i int, ps PeerStore, bd *benchData) error {
		err := ps.DeleteSeeder(bd.infohashes[i%1000], bd.peers[j%1000])
		j += 3
		return err
	})
}

func GradNonexist(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, nil, func(i int, ps PeerStore, bd *benchData) error {
		return ps.GraduateLeecher(bd.infohashes[0], bd.peers[0])
	})
}

func GradNonexist1k(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, nil, func(i int, ps PeerStore, bd *benchData) error {
		return ps.GraduateLeecher(bd.infohashes[0], bd.peers[i%1000])
	})
}

func GradNonexist1kInfohash(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, nil, func(i int, ps PeerStore, bd *benchData) error {
		return ps.GraduateLeecher(bd.infohashes[i%1000], bd.peers[0])
	})
}

func GradNonexist1kInfohash1k(b *testing.B, ps PeerStore) {
	j := 0
	runBenchmark(b, ps, nil, func(i int, ps PeerStore, bd *benchData) error {
		err := ps.GraduateLeecher(bd.infohashes[i%1000], bd.peers[j%1000])
		j += 3
		return err
	})
}

func GradDelete(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, nil, func(i int, ps PeerStore, bd *benchData) error {
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

func GradDelete1k(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, nil, func(i int, ps PeerStore, bd *benchData) error {
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

func GradDelete1kInfohash(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, nil, func(i int, ps PeerStore, bd *benchData) error {
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

func GradDelete1kInfohash1k(b *testing.B, ps PeerStore) {
	j := 0
	runBenchmark(b, ps, nil, func(i int, ps PeerStore, bd *benchData) error {
		err := ps.PutLeecher(bd.infohashes[i%1000], bd.peers[j%1000])
		if err != nil {
			return err
		}
		err = ps.GraduateLeecher(bd.infohashes[i%1000], bd.peers[j%1000])
		if err != nil {
			return err
		}
		err = ps.DeleteSeeder(bd.infohashes[i%1000], bd.peers[j%1000])
		j += 3
		return err
	})
}

func generateAnnounceData(ps PeerStore, bd *benchData) error {
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
	runBenchmark(b, ps, generateAnnounceData, func(i int, ps PeerStore, bd *benchData) error {
		_, err := ps.AnnouncePeers(bd.infohashes[0], false, 50, bd.peers[0])
		return err
	})
}

func AnnounceLeecher1kInfohash(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, generateAnnounceData, func(i int, ps PeerStore, bd *benchData) error {
		_, err := ps.AnnouncePeers(bd.infohashes[i%1000], false, 50, bd.peers[0])
		return err
	})
}

func AnnounceSeeder(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, generateAnnounceData, func(i int, ps PeerStore, bd *benchData) error {
		_, err := ps.AnnouncePeers(bd.infohashes[0], true, 50, bd.peers[0])
		return err
	})
}

func AnnounceSeeder1kInfohash(b *testing.B, ps PeerStore) {
	runBenchmark(b, ps, generateAnnounceData, func(i int, ps PeerStore, bd *benchData) error {
		_, err := ps.AnnouncePeers(bd.infohashes[i%1000], true, 50, bd.peers[0])
		return err
	})
}
