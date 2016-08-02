// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package memory

import (
	"testing"
	"time"

	"github.com/chihaya/chihaya/server/store"
	"github.com/stretchr/testify/require"
)

var (
	peerStoreTester      = store.PreparePeerStoreTester(&peerStoreDriver{})
	peerStoreBenchmarker = store.PreparePeerStoreBenchmarker(&peerStoreDriver{})
	peerStoreTestConfig  = &store.DriverConfig{Config: peerStoreConfig{Shards: 1, GCCutoff: time.Duration(100000000000), GCInterval: time.Duration(100000000000)}}
)

func TestNewPeerStoreConfig(t *testing.T) {
	cfg, err := newPeerStoreConfig(peerStoreTestConfig)
	require.Nil(t, err)
	require.NotNil(t, cfg)

	cfg, err = newPeerStoreConfig(nil)
	require.Equal(t, ErrMissingConfig, err)
	require.Nil(t, cfg)

	cfg, err = newPeerStoreConfig(&store.DriverConfig{})
	require.Equal(t, ErrMissingConfig, err)
	require.Nil(t, cfg)

	cfg, err = newPeerStoreConfig(&store.DriverConfig{Config: nil})
	require.Equal(t, ErrMissingConfig, err)
	require.Nil(t, cfg)

	cfg, err = newPeerStoreConfig(&store.DriverConfig{Config: peerStoreConfig{GCCutoff: time.Duration(50)}})
	require.Equal(t, ErrInvalidGCInterval, err)
	require.Nil(t, cfg)

	cfg, err = newPeerStoreConfig(&store.DriverConfig{Config: peerStoreConfig{GCInterval: time.Duration(50)}})
	require.Equal(t, ErrInvalidGCCutoff, err)
	require.Nil(t, cfg)

	bogus := struct {
		GCInterval string `yaml:"gc_interval"`
		GCCutoff   string `yaml:"gc_cutoff"`
	}{"invalid", "values"}

	cfg, err = newPeerStoreConfig(&store.DriverConfig{Config: bogus})
	require.NotNil(t, err)
	require.Nil(t, cfg)
}

func TestPeerStore(t *testing.T) {
	peerStoreTester.TestPeerStore(t, peerStoreTestConfig)
}

func BenchmarkPeerStore_PutSeeder(b *testing.B) {
	peerStoreBenchmarker.PutSeeder(b, peerStoreTestConfig)
}

func BenchmarkPeerStore_PutSeeder1KInfohash(b *testing.B) {
	peerStoreBenchmarker.PutSeeder1KInfohash(b, peerStoreTestConfig)
}

func BenchmarkPeerStore_PutSeeder1KSeeders(b *testing.B) {
	peerStoreBenchmarker.PutSeeder1KSeeders(b, peerStoreTestConfig)
}

func BenchmarkPeerStore_PutSeeder1KInfohash1KSeeders(b *testing.B) {
	peerStoreBenchmarker.PutSeeder1KInfohash1KSeeders(b, peerStoreTestConfig)
}

func BenchmarkPeerStore_PutDeleteSeeder(b *testing.B) {
	peerStoreBenchmarker.PutDeleteSeeder(b, peerStoreTestConfig)
}

func BenchmarkPeerStore_PutDeleteSeeder1KInfohash(b *testing.B) {
	peerStoreBenchmarker.PutDeleteSeeder1KInfohash(b, peerStoreTestConfig)
}

func BenchmarkPeerStore_PutDeleteSeeder1KSeeders(b *testing.B) {
	peerStoreBenchmarker.PutDeleteSeeder1KSeeders(b, peerStoreTestConfig)
}

func BenchmarkPeerStore_PutDeleteSeeder1KInfohash1KSeeders(b *testing.B) {
	peerStoreBenchmarker.PutDeleteSeeder1KInfohash1KSeeders(b, peerStoreTestConfig)
}

func BenchmarkPeerStore_DeleteSeederNonExist(b *testing.B) {
	peerStoreBenchmarker.DeleteSeederNonExist(b, peerStoreTestConfig)
}

func BenchmarkPeerStore_DeleteSeederNonExist1KInfohash(b *testing.B) {
	peerStoreBenchmarker.DeleteSeederNonExist1KInfohash(b, peerStoreTestConfig)
}

func BenchmarkPeerStore_DeleteSeederNonExist1KSeeders(b *testing.B) {
	peerStoreBenchmarker.DeleteSeederNonExist1KSeeders(b, peerStoreTestConfig)
}

func BenchmarkPeerStore_DeleteSeederNonExist1KInfohash1KSeeders(b *testing.B) {
	peerStoreBenchmarker.DeleteSeederNonExist1KInfohash1KSeeders(b, peerStoreTestConfig)
}

func BenchmarkPeerStore_PutGraduateDeleteLeecher(b *testing.B) {
	peerStoreBenchmarker.PutGraduateDeleteLeecher(b, peerStoreTestConfig)
}

func BenchmarkPeerStore_PutGraduateDeleteLeecher1KInfohash(b *testing.B) {
	peerStoreBenchmarker.PutGraduateDeleteLeecher1KInfohash(b, peerStoreTestConfig)
}

func BenchmarkPeerStore_PutGraduateDeleteLeecher1KSeeders(b *testing.B) {
	peerStoreBenchmarker.PutGraduateDeleteLeecher1KLeechers(b, peerStoreTestConfig)
}

func BenchmarkPeerStore_PutGraduateDeleteLeecher1KInfohash1KSeeders(b *testing.B) {
	peerStoreBenchmarker.PutGraduateDeleteLeecher1KInfohash1KLeechers(b, peerStoreTestConfig)
}

func BenchmarkPeerStore_GraduateLeecherNonExist(b *testing.B) {
	peerStoreBenchmarker.GraduateLeecherNonExist(b, peerStoreTestConfig)
}

func BenchmarkPeerStore_GraduateLeecherNonExist1KInfohash(b *testing.B) {
	peerStoreBenchmarker.GraduateLeecherNonExist1KInfohash(b, peerStoreTestConfig)
}

func BenchmarkPeerStore_GraduateLeecherNonExist1KSeeders(b *testing.B) {
	peerStoreBenchmarker.GraduateLeecherNonExist1KLeechers(b, peerStoreTestConfig)
}

func BenchmarkPeerStore_GraduateLeecherNonExist1KInfohash1KSeeders(b *testing.B) {
	peerStoreBenchmarker.GraduateLeecherNonExist1KInfohash1KLeechers(b, peerStoreTestConfig)
}

func BenchmarkPeerStore_AnnouncePeers(b *testing.B) {
	peerStoreBenchmarker.AnnouncePeers(b, peerStoreTestConfig)
}

func BenchmarkPeerStore_AnnouncePeers1KInfohash(b *testing.B) {
	peerStoreBenchmarker.AnnouncePeers1KInfohash(b, peerStoreTestConfig)
}

func BenchmarkPeerStore_AnnouncePeersSeeder(b *testing.B) {
	peerStoreBenchmarker.AnnouncePeersSeeder(b, peerStoreTestConfig)
}

func BenchmarkPeerStore_AnnouncePeersSeeder1KInfohash(b *testing.B) {
	peerStoreBenchmarker.AnnouncePeersSeeder1KInfohash(b, peerStoreTestConfig)
}

func BenchmarkPeerStore_GetSeeders(b *testing.B) {
	peerStoreBenchmarker.GetSeeders(b, peerStoreTestConfig)
}

func BenchmarkPeerStore_GetSeeders1KInfohash(b *testing.B) {
	peerStoreBenchmarker.GetSeeders1KInfohash(b, peerStoreTestConfig)
}

func BenchmarkPeerStore_NumSeeders(b *testing.B) {
	peerStoreBenchmarker.NumSeeders(b, peerStoreTestConfig)
}

func BenchmarkPeerStore_NumSeeders1KInfohash(b *testing.B) {
	peerStoreBenchmarker.NumSeeders1KInfohash(b, peerStoreTestConfig)
}
