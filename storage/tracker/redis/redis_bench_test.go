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
	conn := createTestConn()
	testUser := createTestUser()
	panicOnErr(conn.AddUser(testUser))
	b.StartTimer()

	for bCount := 0; bCount < b.N; bCount++ {

		foundUser, found, err := conn.FindUser(testUser.Passkey)
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
	panicOnErr(conn.RemoveUser(testUser))
	b.StartTimer()
}

func BenchmarkFailedFindUser(b *testing.B) {
	b.StopTimer()
	conn := createTestConn()
	testUser := createTestUser()
	b.StartTimer()

	for bCount := 0; bCount < b.N; bCount++ {

		_, found, err := conn.FindUser(testUser.Passkey)
		panicOnErr(err)
		if found {
			b.Error("user not found", testUser)
		}
	}
}

func BenchmarkSuccessfulFindTorrent(b *testing.B) {
	b.StopTimer()
	conn := createTestConn()
	testTorrent := createTestTorrent()

	panicOnErr(conn.AddTorrent(testTorrent))
	b.StartTimer()

	for bCount := 0; bCount < b.N; bCount++ {
		foundTorrent, found, err := conn.FindTorrent(testTorrent.Infohash)
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
	panicOnErr(conn.RemoveTorrent(testTorrent))
	b.StartTimer()
}

func BenchmarkFailFindTorrent(b *testing.B) {
	b.StopTimer()
	conn := createTestConn()
	testTorrent := createTestTorrent()
	b.StartTimer()

	for bCount := 0; bCount < b.N; bCount++ {
		foundTorrent, found, err := conn.FindTorrent(testTorrent.Infohash)
		panicOnErr(err)
		if found {
			b.Error("torrent found", foundTorrent)
		}
	}
}

func BenchmarkSuccessfulClientWhitelisted(b *testing.B) {
	b.StopTimer()
	conn := createTestConn()
	testPeerID := "-lt0D30-"
	panicOnErr(conn.WhitelistClient(testPeerID))
	b.StartTimer()

	for bCount := 0; bCount < b.N; bCount++ {
		found, err := conn.ClientWhitelisted(testPeerID)
		panicOnErr(err)
		if !found {
			b.Error("peerID not found", testPeerID)
		}
	}
	// Cleanup
	b.StopTimer()
	panicOnErr(conn.UnWhitelistClient(testPeerID))
	b.StartTimer()
}

func BenchmarkFailClientWhitelisted(b *testing.B) {
	b.StopTimer()
	conn := createTestConn()
	testPeerID2 := "TIX0192"
	b.StartTimer()

	for bCount := 0; bCount < b.N; bCount++ {
		found, err := conn.ClientWhitelisted(testPeerID2)
		panicOnErr(err)
		if found {
			b.Error("peerID found", testPeerID2)
		}
	}
}

func BenchmarkRecordSnatch(b *testing.B) {
	b.StopTimer()
	conn := createTestConn()
	testTorrent := createTestTorrent()
	testUser := createTestUser()
	panicOnErr(conn.AddTorrent(testTorrent))
	panicOnErr(conn.AddUser(testUser))
	b.StartTimer()

	for bCount := 0; bCount < b.N; bCount++ {
		panicOnErr(conn.RecordSnatch(testUser, testTorrent))
	}
	// Cleanup
	b.StopTimer()
	panicOnErr(conn.RemoveTorrent(testTorrent))
	panicOnErr(conn.RemoveUser(testUser))
	b.StartTimer()
}

func BenchmarkMarkActive(b *testing.B) {
	b.StopTimer()
	conn := createTestConn()
	testTorrent := createTestTorrent()
	testTorrent.Active = false
	panicOnErr(conn.AddTorrent(testTorrent))
	b.StartTimer()

	for bCount := 0; bCount < b.N; bCount++ {
		panicOnErr(conn.MarkActive(testTorrent))
	}
	// Cleanup
	b.StopTimer()
	panicOnErr(conn.RemoveTorrent(testTorrent))
	b.StartTimer()
}

func BenchmarkAddSeeder(b *testing.B) {
	b.StopTimer()
	conn := createTestConn()
	testTorrent := createTestTorrent()
	panicOnErr(conn.AddTorrent(testTorrent))
	b.StartTimer()

	for bCount := 0; bCount < b.N; bCount++ {
		b.StopTimer()
		testSeeder := createTestPeer(createTestUserID(), testTorrent.ID)
		b.StartTimer()

		panicOnErr(conn.AddSeeder(testTorrent, testSeeder))
	}
	// Cleanup
	b.StopTimer()
	panicOnErr(conn.RemoveTorrent(testTorrent))
	b.StartTimer()
}

func BenchmarkRemoveSeeder(b *testing.B) {
	b.StopTimer()
	conn := createTestConn()
	testTorrent := createTestTorrent()
	panicOnErr(conn.AddTorrent(testTorrent))
	testSeeder := createTestPeer(createTestUserID(), testTorrent.ID)
	b.StartTimer()

	for bCount := 0; bCount < b.N; bCount++ {
		b.StopTimer()
		conn.AddSeeder(testTorrent, testSeeder)
		b.StartTimer()

		panicOnErr(conn.RemoveSeeder(testTorrent, testSeeder))
	}
	// Cleanup
	b.StopTimer()
	panicOnErr(conn.RemoveTorrent(testTorrent))
	b.StartTimer()
}

func BenchmarkSetSeeder(b *testing.B) {
	b.StopTimer()
	conn := createTestConn()
	testTorrent := createTestTorrent()
	panicOnErr(conn.AddTorrent(testTorrent))
	testSeeder := createTestPeer(createTestUserID(), testTorrent.ID)
	panicOnErr(conn.AddSeeder(testTorrent, testSeeder))
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	b.StartTimer()

	for bCount := 0; bCount < b.N; bCount++ {
		b.StopTimer()
		testSeeder.Uploaded += uint64(r.Int63())
		b.StartTimer()

		conn.SetSeeder(testTorrent, testSeeder)
	}
	// Cleanup
	b.StopTimer()
	panicOnErr(conn.RemoveTorrent(testTorrent))
	b.StartTimer()
}

func BenchmarkIncrementSlots(b *testing.B) {
	b.StopTimer()
	conn := createTestConn()
	testUser := createTestUser()
	panicOnErr(conn.AddUser(testUser))
	b.StartTimer()

	for bCount := 0; bCount < b.N; bCount++ {
		panicOnErr(conn.IncrementSlots(testUser))
	}
	// Cleanup
	b.StopTimer()
	panicOnErr(conn.RemoveUser(testUser))
	b.StartTimer()
}

func BenchmarkLeecherFinished(b *testing.B) {
	b.StopTimer()
	conn := createTestConn()
	testTorrent := createTestTorrent()
	panicOnErr(conn.AddTorrent(testTorrent))
	b.StartTimer()

	for bCount := 0; bCount < b.N; bCount++ {
		b.StopTimer()
		testLeecher := createTestPeer(createTestUserID(), testTorrent.ID)
		panicOnErr(conn.AddLeecher(testTorrent, testLeecher))
		testLeecher.Left = 0
		b.StartTimer()

		panicOnErr(conn.LeecherFinished(testTorrent, testLeecher))
	}
	// Cleanup
	b.StopTimer()
	panicOnErr(conn.RemoveTorrent(testTorrent))
	b.StartTimer()
}

// This is a comparision to the Leecher finished function
func BenchmarkRemoveLeecherAddSeeder(b *testing.B) {
	b.StopTimer()
	conn := createTestConn()
	testTorrent := createTestTorrent()
	panicOnErr(conn.AddTorrent(testTorrent))
	b.StartTimer()

	for bCount := 0; bCount < b.N; bCount++ {
		b.StopTimer()
		testLeecher := createTestPeer(createTestUserID(), testTorrent.ID)
		panicOnErr(conn.AddLeecher(testTorrent, testLeecher))
		testLeecher.Left = 0
		b.StartTimer()

		panicOnErr(conn.RemoveLeecher(testTorrent, testLeecher))
		panicOnErr(conn.AddSeeder(testTorrent, testLeecher))
	}
	// Cleanup
	b.StopTimer()
	conn.RemoveTorrent(testTorrent)
	b.StartTimer()
}
