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

func BenchmarkNop(b *testing.B)                        { s.Nop(b, createNew()) }
func BenchmarkPut(b *testing.B)                        { s.Put(b, createNew()) }
func BenchmarkPut1k(b *testing.B)                      { s.Put1k(b, createNew()) }
func BenchmarkPut1kInfohash(b *testing.B)              { s.Put1kInfohash(b, createNew()) }
func BenchmarkPut1kInfohash1k(b *testing.B)            { s.Put1kInfohash1k(b, createNew()) }
func BenchmarkPutDelete(b *testing.B)                  { s.PutDelete(b, createNew()) }
func BenchmarkPutDelete1k(b *testing.B)                { s.PutDelete1k(b, createNew()) }
func BenchmarkPutDelete1kInfohash(b *testing.B)        { s.PutDelete1kInfohash(b, createNew()) }
func BenchmarkPutDelete1kInfohash1k(b *testing.B)      { s.PutDelete1kInfohash1k(b, createNew()) }
func BenchmarkDeleteNonexist(b *testing.B)             { s.DeleteNonexist(b, createNew()) }
func BenchmarkDeleteNonexist1k(b *testing.B)           { s.DeleteNonexist1k(b, createNew()) }
func BenchmarkDeleteNonexist1kInfohash(b *testing.B)   { s.DeleteNonexist1kInfohash(b, createNew()) }
func BenchmarkDeleteNonexist1kInfohash1k(b *testing.B) { s.DeleteNonexist1kInfohash1k(b, createNew()) }
func BenchmarkPutGradDelete(b *testing.B)              { s.PutGradDelete(b, createNew()) }
func BenchmarkPutGradDelete1k(b *testing.B)            { s.PutGradDelete1k(b, createNew()) }
func BenchmarkPutGradDelete1kInfohash(b *testing.B)    { s.PutGradDelete1kInfohash(b, createNew()) }
func BenchmarkPutGradDelete1kInfohash1k(b *testing.B)  { s.PutGradDelete1kInfohash1k(b, createNew()) }
func BenchmarkGradNonexist(b *testing.B)               { s.GradNonexist(b, createNew()) }
func BenchmarkGradNonexist1k(b *testing.B)             { s.GradNonexist1k(b, createNew()) }
func BenchmarkGradNonexist1kInfohash(b *testing.B)     { s.GradNonexist1kInfohash(b, createNew()) }
func BenchmarkGradNonexist1kInfohash1k(b *testing.B)   { s.GradNonexist1kInfohash1k(b, createNew()) }
func BenchmarkAnnounceLeecherLarge(b *testing.B)       { s.AnnounceLeecherLarge(b, createNew()) }
func BenchmarkAnnounceLeecherLarge1kInfohash(b *testing.B) {
	s.AnnounceLeecherLarge1kInfohash(b, createNew())
}
func BenchmarkAnnounceSeederLarge(b *testing.B) { s.AnnounceSeederLarge(b, createNew()) }
func BenchmarkAnnounceSeederLarge1kInfohash(b *testing.B) {
	s.AnnounceSeederLarge1kInfohash(b, createNew())
}
func BenchmarkAnnounceLeecherSmall(b *testing.B) { s.AnnounceLeecherSmall(b, createNew()) }
func BenchmarkAnnounceLeecherSmall1kInfohash(b *testing.B) {
	s.AnnounceLeecherSmall1kInfohash(b, createNew())
}
func BenchmarkAnnounceSeederSmall(b *testing.B) { s.AnnounceSeederSmall(b, createNew()) }
func BenchmarkAnnounceSeederSmall1kInfohash(b *testing.B) {
	s.AnnounceSeederSmall1kInfohash(b, createNew())
}
func BenchmarkScrapeSwarmLarge(b *testing.B)           { s.ScrapeSwarmLarge(b, createNew()) }
func BenchmarkScrapeSwarmLarge1kInfohash(b *testing.B) { s.ScrapeSwarmLarge1kInfohash(b, createNew()) }
func BenchmarkScrapeSwarmSmall(b *testing.B)           { s.ScrapeSwarmSmall(b, createNew()) }
func BenchmarkScrapeSwarmSmall1kInfohash(b *testing.B) { s.ScrapeSwarmSmall1kInfohash(b, createNew()) }
