package memory

import (
	"testing"
	"time"

	s "github.com/chihaya/chihaya/storage"
)

func createNew() s.ClearablePeerStore {
	ps, err := build(Config{
		ShardCount:                  1024,
		GarbageCollectionInterval:   10 * time.Minute,
		PrometheusReportingInterval: 10 * time.Minute,
		PeerLifetime:                30 * time.Minute,
	})
	if err != nil {
		panic(err)
	}
	return ps
}

func TestPeerStore(t *testing.T) { s.TestPeerStore(t, createNew()) }

func BenchmarkNop(b *testing.B)                         { s.Nop(b, createNew()) }
func BenchmarkPut(b *testing.B)                         { s.Put(b, createNew()) }
func BenchmarkPutSpreadPeer(b *testing.B)               { s.PutSpreadPeer(b, createNew()) }
func BenchmarkPutSpreadInfohash(b *testing.B)           { s.PutSpreadInfohash(b, createNew()) }
func BenchmarkPutSpreadInfohashSpreadPeer(b *testing.B) { s.PutSpreadInfohashSpreadPeer(b, createNew()) }
func BenchmarkPutDelete(b *testing.B)                   { s.PutDelete(b, createNew()) }
func BenchmarkPutDeleteSpreadPeer(b *testing.B)         { s.PutDeleteSpreadPeer(b, createNew()) }
func BenchmarkPutDeleteSpreadInfohash(b *testing.B)     { s.PutDeleteSpreadInfohash(b, createNew()) }
func BenchmarkPutDeleteSpreadInfohashSpreadPeer(b *testing.B) {
	s.PutDeleteSpreadInfohashSpreadPeer(b, createNew())
}
func BenchmarkDeleteNonexist(b *testing.B)           { s.DeleteNonexist(b, createNew()) }
func BenchmarkDeleteNonexistSpreadPeer(b *testing.B) { s.DeleteNonexistSpreadPeer(b, createNew()) }
func BenchmarkDeleteNonexistSpreadInfohash(b *testing.B) {
	s.DeleteNonexistSpreadInfohash(b, createNew())
}
func BenchmarkDeleteNonexistSpreadInfohashSpreadPeer(b *testing.B) {
	s.DeleteNonexistSpreadInfohashSpreadPeer(b, createNew())
}
func BenchmarkPutGradDelete(b *testing.B)               { s.PutGradDelete(b, createNew()) }
func BenchmarkPutGradDeleteSpreadPeer(b *testing.B)     { s.PutGradDeleteSpreadPeer(b, createNew()) }
func BenchmarkPutGradDeleteSpreadInfohash(b *testing.B) { s.PutGradDeleteSpreadInfohash(b, createNew()) }
func BenchmarkPutGradDeleteSpreadInfohashSpreadPeer(b *testing.B) {
	s.PutGradDeleteSpreadInfohashSpreadPeer(b, createNew())
}
func BenchmarkGradNonexist(b *testing.B)               { s.GradNonexist(b, createNew()) }
func BenchmarkGradNonexistSpreadPeer(b *testing.B)     { s.GradNonexistSpreadPeer(b, createNew()) }
func BenchmarkGradNonexistSpreadInfohash(b *testing.B) { s.GradNonexistSpreadInfohash(b, createNew()) }
func BenchmarkGradNonexistSpreadInfohashSpreadPeer(b *testing.B) {
	s.GradNonexistSpreadInfohashSpreadPeer(b, createNew())
}
func BenchmarkAnnounceLeecherLarge(b *testing.B) { s.AnnounceLeecherLarge(b, createNew()) }
func BenchmarkAnnounceLeecherLargeSpreadInfohash(b *testing.B) {
	s.AnnounceLeecherLargeSpreadInfohash(b, createNew())
}
func BenchmarkAnnounceSeederLarge(b *testing.B) { s.AnnounceSeederLarge(b, createNew()) }
func BenchmarkAnnounceSeederLargeSpreadInfohash(b *testing.B) {
	s.AnnounceSeederLargeSpreadInfohash(b, createNew())
}
func BenchmarkAnnounceLeecherSmall(b *testing.B) { s.AnnounceLeecherSmall(b, createNew()) }
func BenchmarkAnnounceLeecherSmallSpreadInfohash(b *testing.B) {
	s.AnnounceLeecherSmallSpreadInfohash(b, createNew())
}
func BenchmarkAnnounceSeederSmall(b *testing.B) { s.AnnounceSeederSmall(b, createNew()) }
func BenchmarkAnnounceSeederSmallSpreadInfohash(b *testing.B) {
	s.AnnounceSeederSmallSpreadInfohash(b, createNew())
}
func BenchmarkScrapeSwarmLarge(b *testing.B) { s.ScrapeSwarmLarge(b, createNew()) }
func BenchmarkScrapeSwarmLargeSpreadInfohash(b *testing.B) {
	s.ScrapeSwarmLargeSpreadInfohash(b, createNew())
}
func BenchmarkScrapeSwarmSmall(b *testing.B) { s.ScrapeSwarmSmall(b, createNew()) }
func BenchmarkScrapeSwarmSmallSpreadInfohash(b *testing.B) {
	s.ScrapeSwarmSmallSpreadInfohash(b, createNew())
}
