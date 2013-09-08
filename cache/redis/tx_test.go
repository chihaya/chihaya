// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package redis

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/pushrax/chihaya/cache"
	"github.com/pushrax/chihaya/config"
)

func panicErrNil(err error) {
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
}

func createTestTx() cache.Tx {
	testConfig, err := config.Open(os.Getenv("TESTCONFIGPATH"))
	panicErrNil(err)
	conf := &testConfig.Cache

	testPool, err := cache.Open(conf)
	panicErrNil(err)

	txObj, err := testPool.Get()
	panicErrNil(err)

	return txObj
}

func TestUser(t *testing.T) {
	tx := createTestTx()
	testUser1 := createTestUser()
	testUser2 := createTestUser()

	panicErrNil(tx.AddUser(&testUser1))
	foundUser, found, err := tx.FindUser(testUser1.Passkey)
	panicErrNil(err)
	if !found {
		t.Error("user not found")
	}
	if *foundUser != testUser1 {
		t.Error("found user mismatch")
	}

	foundUser, found, err = tx.FindUser(testUser2.Passkey)
	panicErrNil(err)
	if found {
		t.Error("user found")
	}

	err = tx.RemoveUser(&testUser1)
	panicErrNil(err)
	foundUser, found, err = tx.FindUser(testUser1.Passkey)
	panicErrNil(err)
	if found {
		t.Error("removed user found")
	}
}

func TestTorrent(t *testing.T) {
	tx := createTestTx()
	testTorrent1 := createTestTorrent()
	testTorrent2 := createTestTorrent()

	panicErrNil(tx.AddTorrent(&testTorrent1))
	foundTorrent, found, err := tx.FindTorrent(testTorrent1.Infohash)
	panicErrNil(err)
	if !found {
		t.Error("torrent not found")
	}
	// Incomplete comparison as maps cannot be compared
	if foundTorrent.Infohash != testTorrent1.Infohash {
		t.Error("found torrent mismatch")
	}
	foundTorrent, found, err = tx.FindTorrent(testTorrent2.Infohash)
	panicErrNil(err)
	if found {
		t.Error("torrent found")
	}

	panicErrNil(tx.RemoveTorrent(&testTorrent1))
	foundTorrent, found, err = tx.FindTorrent(testTorrent1.Infohash)
	panicErrNil(err)
	if found {
		t.Error("removed torrent found")
	}
}
func TestClient(t *testing.T) {
	tx := createTestTx()
	testPeerID1 := "-lt0D30-"
	testPeerID2 := "TIX0192"

	panicErrNil(tx.WhitelistClient(testPeerID1))
	found, err := tx.ClientWhitelisted(testPeerID1)
	panicErrNil(err)
	if !found {
		t.Error("peerID not found")
	}

	found, err = tx.ClientWhitelisted(testPeerID2)
	panicErrNil(err)
	if found {
		t.Error("peerID found")
	}

	panicErrNil(tx.UnWhitelistClient(testPeerID1))
	found, err = tx.ClientWhitelisted(testPeerID1)
	panicErrNil(err)
	if found {
		t.Error("removed peerID found")
	}
}

func TestPeers(t *testing.T) {
	tx := createTestTx()

	// Randomly generated strings would be safter to test with
	testTorrent1 := createTestTorrent()
	testTorrent2 := createTestTorrent()
	foundTorrent, found, err := tx.FindTorrent(testTorrent1.Infohash)
	panicErrNil(err)
	if found {
		testTorrent1 = *foundTorrent
	} else {
		panicErrNil(tx.AddTorrent(&testTorrent1))
	}
	foundTorrent, found, err = tx.FindTorrent(testTorrent2.Infohash)
	panicErrNil(err)
	if found {
		testTorrent2 = *foundTorrent
	} else {
		panicErrNil(tx.AddTorrent(&testTorrent2))
	}

	testSeeder1 := createTestPeer(createTestUserID(), testTorrent1.ID)
	testSeeder2 := createTestPeer(createTestUserID(), testTorrent2.ID)
	if testSeeder1 == testSeeder2 {
		t.Error("seeders should not be equal")
	}

	if _, exists := testTorrent1.Seeders[testSeeder1.ID]; exists {
		t.Log("seeder aleady exists, removing")
		err := tx.RemoveSeeder(&testTorrent1, &testSeeder1)
		if err != nil {
			t.Error(err)
		}
		if _, exists := testTorrent1.Seeders[testSeeder1.ID]; exists {
			t.Error("Remove seeder failed")
		}
	}

	panicErrNil(tx.AddSeeder(&testTorrent1, &testSeeder1))
	if seeder1, exists := testTorrent1.Seeders[testSeeder1.ID]; !exists {
		t.Error("seeder not added locally")
	} else if seeder1 != testSeeder1 {
		t.Error("seeder changed")
	}
	foundTorrent, found, err = tx.FindTorrent(testTorrent1.Infohash)
	panicErrNil(err)
	if !found {
		t.Error("torrent should exist")
	}
	if seeder1, exists := foundTorrent.Seeders[testSeeder1.ID]; !exists {
		t.Error("seeder not added")
	} else if seeder1 != testSeeder1 {
		t.Error("seeder changed")
	}

	// Update a seeder, set it, then check to make sure it updated
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	testSeeder1.Downloaded += uint64(r.Int63())
	panicErrNil(tx.SetSeeder(&testTorrent1, &testSeeder1))
	foundTorrent, found, err = tx.FindTorrent(testTorrent1.Infohash)
	panicErrNil(err)
	if seeder1, exists := foundTorrent.Seeders[testSeeder1.ID]; !exists {
		t.Error("seeder not added")
	} else if seeder1 != testSeeder1 {
		t.Errorf("seeder changed from %v to %v", testSeeder1, seeder1)
	}

}
