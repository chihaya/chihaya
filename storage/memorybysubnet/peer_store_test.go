package memorybysubnet

import (
	"testing"
	"time"

	s "github.com/chihaya/chihaya/storage"
)

func createNew() s.PeerStore {
	ps, err := New(Config{
		ShardCount:                1024,
		GarbageCollectionInterval: 10 * time.Minute,
	})
	if err != nil {
		panic(err)
	}
	return ps
}

func TestPeerStore(t *testing.T) { s.TestPeerStore(t, createNew()) }

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
func BenchmarkAnnounceLeecher(b *testing.B)            { s.AnnounceLeecher(b, createNew()) }
func BenchmarkAnnounceLeecher1kInfohash(b *testing.B)  { s.AnnounceLeecher1kInfohash(b, createNew()) }
func BenchmarkAnnounceSeeder(b *testing.B)             { s.AnnounceSeeder(b, createNew()) }
func BenchmarkAnnounceSeeder1kInfohash(b *testing.B)   { s.AnnounceSeeder1kInfohash(b, createNew()) }
