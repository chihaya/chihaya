// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package redis

import (
	"math/rand"
	"testing"
	"time"
)

func BenchmarkSuccessfulFindUser(b *testing.B) {
	b.StopTimer()
	tx := createTestTx()
	testUser := createTestUser()
	panicOnErr(tx.AddUser(testUser))
	b.StartTimer()

	for bCount := 0; bCount < b.N; bCount++ {

		foundUser, found, err := tx.FindUser(testUser.Passkey)
		panicOnErr(err)
		if !found {
			b.Error("user not found", testUser)
		}
		if *foundUser != *testUser {
			b.Error("found user mismatch", *foundUser, testUser)
		}
	}
	// Cleanup
	b.StopTimer()
	panicOnErr(tx.RemoveUser(testUser))
	b.StartTimer()
}

func BenchmarkFailedFindUser(b *testing.B) {
	b.StopTimer()
	tx := createTestTx()
	testUser := createTestUser()
	b.StartTimer()

	for bCount := 0; bCount < b.N; bCount++ {

		_, found, err := tx.FindUser(testUser.Passkey)
		panicOnErr(err)
		if found {
			b.Error("user not found", testUser)
		}
	}
}

func BenchmarkSuccessfulFindTorrent(b *testing.B) {
	b.StopTimer()
	tx := createTestTx()
	testTorrent := createTestTorrent()

	panicOnErr(tx.AddTorrent(testTorrent))
	b.StartTimer()

	for bCount := 0; bCount < b.N; bCount++ {
		foundTorrent, found, err := tx.FindTorrent(testTorrent.Infohash)
		panicOnErr(err)
		if !found {
			b.Error("torrent not found", testTorrent)
		}
		// Incomplete comparison as maps make struct not nativly comparable
		if foundTorrent.Infohash != testTorrent.Infohash {
			b.Error("found torrent mismatch", foundTorrent, testTorrent)
		}
	}
	// Cleanup
	b.StopTimer()
	panicOnErr(tx.RemoveTorrent(testTorrent))
	b.StartTimer()
}

func BenchmarkFailFindTorrent(b *testing.B) {
	b.StopTimer()
	tx := createTestTx()
	testTorrent := createTestTorrent()
	b.StartTimer()

	for bCount := 0; bCount < b.N; bCount++ {
		foundTorrent, found, err := tx.FindTorrent(testTorrent.Infohash)
		panicOnErr(err)
		if found {
			b.Error("torrent found", foundTorrent)
		}
	}
}

func BenchmarkSuccessfulClientWhitelisted(b *testing.B) {
	b.StopTimer()
	tx := createTestTx()
	testPeerID := "-lt0D30-"
	panicOnErr(tx.WhitelistClient(testPeerID))
	b.StartTimer()

	for bCount := 0; bCount < b.N; bCount++ {
		found, err := tx.ClientWhitelisted(testPeerID)
		panicOnErr(err)
		if !found {
			b.Error("peerID not found", testPeerID)
		}
	}
	// Cleanup
	b.StopTimer()
	panicOnErr(tx.UnWhitelistClient(testPeerID))
	b.StartTimer()
}

func BenchmarkFailClientWhitelisted(b *testing.B) {
	b.StopTimer()
	tx := createTestTx()
	testPeerID2 := "TIX0192"
	b.StartTimer()

	for bCount := 0; bCount < b.N; bCount++ {
		found, err := tx.ClientWhitelisted(testPeerID2)
		panicOnErr(err)
		if found {
			b.Error("peerID found", testPeerID2)
		}
	}
}

func BenchmarkRecordSnatch(b *testing.B) {
	b.StopTimer()
	tx := createTestTx()
	testTorrent := createTestTorrent()
	testUser := createTestUser()
	panicOnErr(tx.AddTorrent(testTorrent))
	panicOnErr(tx.AddUser(testUser))
	b.StartTimer()

	for bCount := 0; bCount < b.N; bCount++ {
		panicOnErr(tx.RecordSnatch(testUser, testTorrent))
	}
	// Cleanup
	b.StopTimer()
	panicOnErr(tx.RemoveTorrent(testTorrent))
	panicOnErr(tx.RemoveUser(testUser))
	b.StartTimer()
}

func BenchmarkMarkActive(b *testing.B) {
	b.StopTimer()
	tx := createTestTx()
	testTorrent := createTestTorrent()
	testTorrent.Active = false
	panicOnErr(tx.AddTorrent(testTorrent))
	b.StartTimer()

	for bCount := 0; bCount < b.N; bCount++ {
		panicOnErr(tx.MarkActive(testTorrent))
	}
	// Cleanup
	b.StopTimer()
	panicOnErr(tx.RemoveTorrent(testTorrent))
	b.StartTimer()
}

func BenchmarkAddSeeder(b *testing.B) {
	b.StopTimer()
	tx := createTestTx()
	testTorrent := createTestTorrent()
	panicOnErr(tx.AddTorrent(testTorrent))
	b.StartTimer()

	for bCount := 0; bCount < b.N; bCount++ {
		b.StopTimer()
		testSeeder := createTestPeer(createTestUserID(), testTorrent.ID)
		b.StartTimer()

		panicOnErr(tx.AddSeeder(testTorrent, testSeeder))
	}
	// Cleanup
	b.StopTimer()
	panicOnErr(tx.RemoveTorrent(testTorrent))
	b.StartTimer()
}

func BenchmarkRemoveSeeder(b *testing.B) {
	b.StopTimer()
	tx := createTestTx()
	testTorrent := createTestTorrent()
	panicOnErr(tx.AddTorrent(testTorrent))
	testSeeder := createTestPeer(createTestUserID(), testTorrent.ID)
	b.StartTimer()

	for bCount := 0; bCount < b.N; bCount++ {
		b.StopTimer()
		tx.AddSeeder(testTorrent, testSeeder)
		b.StartTimer()

		panicOnErr(tx.RemoveSeeder(testTorrent, testSeeder))
	}
	// Cleanup
	b.StopTimer()
	panicOnErr(tx.RemoveTorrent(testTorrent))
	b.StartTimer()
}

func BenchmarkSetSeeder(b *testing.B) {
	b.StopTimer()
	tx := createTestTx()
	testTorrent := createTestTorrent()
	panicOnErr(tx.AddTorrent(testTorrent))
	testSeeder := createTestPeer(createTestUserID(), testTorrent.ID)
	panicOnErr(tx.AddSeeder(testTorrent, testSeeder))
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	b.StartTimer()

	for bCount := 0; bCount < b.N; bCount++ {
		b.StopTimer()
		testSeeder.Uploaded += uint64(r.Int63())
		b.StartTimer()

		tx.SetSeeder(testTorrent, testSeeder)
	}
	// Cleanup
	b.StopTimer()
	panicOnErr(tx.RemoveTorrent(testTorrent))
	b.StartTimer()
}

func BenchmarkIncrementSlots(b *testing.B) {
	b.StopTimer()
	tx := createTestTx()
	testUser := createTestUser()
	panicOnErr(tx.AddUser(testUser))
	b.StartTimer()

	for bCount := 0; bCount < b.N; bCount++ {
		panicOnErr(tx.IncrementSlots(testUser))
	}
	// Cleanup
	b.StopTimer()
	panicOnErr(tx.RemoveUser(testUser))
	b.StartTimer()
}

func BenchmarkLeecherFinished(b *testing.B) {
	b.StopTimer()
	tx := createTestTx()
	testTorrent := createTestTorrent()
	panicOnErr(tx.AddTorrent(testTorrent))
	b.StartTimer()

	for bCount := 0; bCount < b.N; bCount++ {
		b.StopTimer()
		testLeecher := createTestPeer(createTestUserID(), testTorrent.ID)
		panicOnErr(tx.AddLeecher(testTorrent, testLeecher))
		testLeecher.Left = 0
		b.StartTimer()

		panicOnErr(tx.LeecherFinished(testTorrent, testLeecher))
	}
	// Cleanup
	b.StopTimer()
	panicOnErr(tx.RemoveTorrent(testTorrent))
	b.StartTimer()
}

// This is a comparision to the Leecher finished function
func BenchmarkRemoveLeecherAddSeeder(b *testing.B) {
	b.StopTimer()
	tx := createTestTx()
	testTorrent := createTestTorrent()
	panicOnErr(tx.AddTorrent(testTorrent))
	b.StartTimer()

	for bCount := 0; bCount < b.N; bCount++ {
		b.StopTimer()
		testLeecher := createTestPeer(createTestUserID(), testTorrent.ID)
		panicOnErr(tx.AddLeecher(testTorrent, testLeecher))
		testLeecher.Left = 0
		b.StartTimer()

		panicOnErr(tx.RemoveLeecher(testTorrent, testLeecher))
		panicOnErr(tx.AddSeeder(testTorrent, testLeecher))
	}
	// Cleanup
	b.StopTimer()
	tx.RemoveTorrent(testTorrent)
	b.StartTimer()
}
