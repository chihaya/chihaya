package memory

import (
	"testing"

	s "github.com/jzelinskie/trakr/storage"
)

func BenchmarkPut(b *testing.B)                        { s.Put(b, &peerStore{}) }
func BenchmarkPut1k(b *testing.B)                      { s.Put1k(b, &peerStore{}) }
func BenchmarkPut1kInfohash(b *testing.B)              { s.Put1kInfohash(b, &peerStore{}) }
func BenchmarkPut1kInfohash1k(b *testing.B)            { s.Put1kInfohash1k(b, &peerStore{}) }
func BenchmarkPutDelete(b *testing.B)                  { s.PutDelete(b, &peerStore{}) }
func BenchmarkPutDelete1k(b *testing.B)                { s.PutDelete1k(b, &peerStore{}) }
func BenchmarkPutDelete1kInfohash(b *testing.B)        { s.PutDelete1kInfohash(b, &peerStore{}) }
func BenchmarkPutDelete1kInfohash1k(b *testing.B)      { s.PutDelete1kInfohash1k(b, &peerStore{}) }
func BenchmarkDeleteNonexist(b *testing.B)             { s.DeleteNonexist(b, &peerStore{}) }
func BenchmarkDeleteNonexist1k(b *testing.B)           { s.DeleteNonexist1k(b, &peerStore{}) }
func BenchmarkDeleteNonexist1kInfohash(b *testing.B)   { s.DeleteNonexist1kInfohash(b, &peerStore{}) }
func BenchmarkDeleteNonexist1kInfohash1k(b *testing.B) { s.DeleteNonexist1kInfohash1k(b, &peerStore{}) }
func BenchmarkGradDelete(b *testing.B)                 { s.GradDelete(b, &peerStore{}) }
func BenchmarkGradDelete1k(b *testing.B)               { s.GradDelete1k(b, &peerStore{}) }
func BenchmarkGradDelete1kInfohash(b *testing.B)       { s.GradDelete1kInfohash(b, &peerStore{}) }
func BenchmarkGradDelete1kInfohash1k(b *testing.B)     { s.GradDelete1kInfohash1k(b, &peerStore{}) }
func BenchmarkGradNonexist(b *testing.B)               { s.GradNonexist(b, &peerStore{}) }
func BenchmarkGradNonexist1k(b *testing.B)             { s.GradNonexist1k(b, &peerStore{}) }
func BenchmarkGradNonexist1kInfohash(b *testing.B)     { s.GradNonexist1kInfohash(b, &peerStore{}) }
func BenchmarkGradNonexist1kInfohash1k(b *testing.B)   { s.GradNonexist1kInfohash1k(b, &peerStore{}) }
func BenchmarkAnnounceLeecher(b *testing.B)            { s.AnnounceLeecher(b, &peerStore{}) }
func BenchmarkAnnounceLeecher1kInfohash(b *testing.B)  { s.AnnounceLeecher1kInfohash(b, &peerStore{}) }
func BenchmarkAnnounceSeeder(b *testing.B)             { s.AnnounceSeeder(b, &peerStore{}) }
func BenchmarkAnnounceSeeder1kInfohash(b *testing.B)   { s.AnnounceSeeder1kInfohash(b, &peerStore{}) }
