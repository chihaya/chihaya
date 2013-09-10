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

func TestFindUserSuccess(t *testing.T) {
	tx := createTestTx()
	testUser1 := createTestUser()

	panicErrNil(tx.AddUser(&testUser1))
	foundUser, found, err := tx.FindUser(testUser1.Passkey)
	panicErrNil(err)
	if !found {
		t.Error("user not found", testUser1)
	}
	if *foundUser != testUser1 {
		t.Error("found user mismatch", *foundUser, testUser1)
	}
}

func TestFindUserFail(t *testing.T) {
	tx := createTestTx()
	testUser2 := createTestUser()

	foundUser, found, err := tx.FindUser(testUser2.Passkey)
	panicErrNil(err)
	if found {
		t.Error("user found", foundUser)
	}
}

func TestRemoveUser(t *testing.T) {
	tx := createTestTx()
	testUser1 := createTestUser()

	panicErrNil(tx.AddUser(&testUser1))
	err := tx.RemoveUser(&testUser1)
	panicErrNil(err)
	foundUser, found, err := tx.FindUser(testUser1.Passkey)
	panicErrNil(err)
	if found {
		t.Error("removed user found", foundUser)
	}
}

func TestFindTorrent(t *testing.T) {
	tx := createTestTx()
	testTorrent1 := createTestTorrent()

	panicErrNil(tx.AddTorrent(testTorrent1))
	foundTorrent, found, err := tx.FindTorrent(testTorrent1.Infohash)
	panicErrNil(err)
	if !found {
		t.Error("torrent not found", testTorrent1)
	}
	// Incomplete comparison as maps make struct not nativly comparable
	if foundTorrent.Infohash != testTorrent1.Infohash {
		t.Error("found torrent mismatch", foundTorrent, testTorrent1)
	}
}

func TestFindTorrentFail(t *testing.T) {
	tx := createTestTx()
	testTorrent2 := createTestTorrent()

	foundTorrent, found, err := tx.FindTorrent(testTorrent2.Infohash)
	panicErrNil(err)
	if found {
		t.Error("torrent found", foundTorrent)
	}
}

func TestRemoveTorrent(t *testing.T) {
	tx := createTestTx()
	testTorrent1 := createTestTorrent()
	panicErrNil(tx.AddTorrent(testTorrent1))

	panicErrNil(tx.RemoveTorrent(testTorrent1))
	foundTorrent, found, err := tx.FindTorrent(testTorrent1.Infohash)
	panicErrNil(err)
	if found {
		t.Error("removed torrent found", foundTorrent)
	}
}

func TestClientWhitelistSuccess(t *testing.T) {
	tx := createTestTx()
	testPeerID1 := "-lt0D30-"

	panicErrNil(tx.WhitelistClient(testPeerID1))
	found, err := tx.ClientWhitelisted(testPeerID1)
	panicErrNil(err)
	if !found {
		t.Error("peerID not found", testPeerID1)
	}
}

func TestClientWhitelistFail(t *testing.T) {
	tx := createTestTx()
	testPeerID2 := "TIX0192"

	found, err := tx.ClientWhitelisted(testPeerID2)
	panicErrNil(err)
	if found {
		t.Error("peerID found", testPeerID2)
	}

}

func TestClientWhitelistRemove(t *testing.T) {
	tx := createTestTx()
	testPeerID1 := "-lt0D30-"
	panicErrNil(tx.WhitelistClient(testPeerID1))
	panicErrNil(tx.UnWhitelistClient(testPeerID1))

	found, err := tx.ClientWhitelisted(testPeerID1)
	panicErrNil(err)
	if found {
		t.Error("removed peerID found", testPeerID1)
	}
}

func TestAddSeeder(t *testing.T) {
	tx := createTestTx()
	testTorrent1 := createTestTorrent()
	panicErrNil(tx.AddTorrent(testTorrent1))
	testSeeder1 := createTestPeer(createTestUserID(), testTorrent1.ID)

	panicErrNil(tx.AddSeeder(testTorrent1, testSeeder1))
	foundTorrent, found, err := tx.FindTorrent(testTorrent1.Infohash)
	panicErrNil(err)
	foundSeeder, found := foundTorrent.Seeders[testSeeder1.ID]
	if found && foundSeeder != *testSeeder1 {
		t.Error("seeder not added to cache", testSeeder1)
	}
	foundSeeder, found = testTorrent1.Seeders[testSeeder1.ID]
	if found && foundSeeder != *testSeeder1 {
		t.Error("seeder not added to local", testSeeder1)
	}
}

func TestAddLeecher(t *testing.T) {
	tx := createTestTx()
	testTorrent1 := createTestTorrent()
	panicErrNil(tx.AddTorrent(testTorrent1))
	testLeecher1 := createTestPeer(createTestUserID(), testTorrent1.ID)

	tx.AddLeecher(testTorrent1, testLeecher1)
	foundTorrent, found, err := tx.FindTorrent(testTorrent1.Infohash)
	panicErrNil(err)
	foundLeecher, found := foundTorrent.Leechers[testLeecher1.ID]
	if found && foundLeecher != *testLeecher1 {
		t.Error("leecher not added to cache", testLeecher1)
	}
	foundLeecher, found = testTorrent1.Leechers[testLeecher1.ID]
	if found && foundLeecher != *testLeecher1 {
		t.Error("leecher not added to local", testLeecher1)
	}
}

func TestRemoveSeeder(t *testing.T) {
	tx := createTestTx()
	testTorrent1 := createTestTorrent()
	panicErrNil(tx.AddTorrent(testTorrent1))
	testSeeder1 := createTestPeer(createTestUserID(), testTorrent1.ID)
	tx.AddSeeder(testTorrent1, testSeeder1)

	panicErrNil(tx.RemoveSeeder(testTorrent1, testSeeder1))
	foundSeeder, found := testTorrent1.Seeders[testSeeder1.ID]
	if found || foundSeeder == *testSeeder1 {
		t.Error("seeder not removed from local", foundSeeder)
	}

	foundTorrent, found, err := tx.FindTorrent(testTorrent1.Infohash)
	panicErrNil(err)
	foundSeeder, found = foundTorrent.Seeders[testSeeder1.ID]
	if found || foundSeeder == *testSeeder1 {
		t.Error("seeder not removed from cache", foundSeeder)
	}
}

func TestRemoveLeecher(t *testing.T) {
	tx := createTestTx()
	testTorrent1 := createTestTorrent()
	panicErrNil(tx.AddTorrent(testTorrent1))
	testLeecher1 := createTestPeer(createTestUserID(), testTorrent1.ID)
	tx.AddLeecher(testTorrent1, testLeecher1)

	tx.RemoveLeecher(testTorrent1, testLeecher1)
	foundTorrent, found, err := tx.FindTorrent(testTorrent1.Infohash)
	panicErrNil(err)
	foundLeecher, found := foundTorrent.Leechers[testLeecher1.ID]
	if found || foundLeecher == *testLeecher1 {
		t.Error("leecher not removed from cache", foundLeecher)
	}
	foundLeecher, found = testTorrent1.Leechers[testLeecher1.ID]
	if found || foundLeecher == *testLeecher1 {
		t.Error("leecher not removed from local", foundLeecher)
	}
}

func TestSetSeeder(t *testing.T) {
	tx := createTestTx()
	testTorrent1 := createTestTorrent()
	panicErrNil(tx.AddTorrent(testTorrent1))
	testSeeder1 := createTestPeer(createTestUserID(), testTorrent1.ID)
	tx.AddSeeder(testTorrent1, testSeeder1)

	testSeeder1.Uploaded += 100

	tx.SetSeeder(testTorrent1, testSeeder1)
	foundTorrent, _, err := tx.FindTorrent(testTorrent1.Infohash)
	panicErrNil(err)
	foundSeeder, _ := foundTorrent.Seeders[testSeeder1.ID]
	if foundSeeder != *testSeeder1 {
		t.Error("seeder not updated in cache", testSeeder1)
	}
	foundSeeder, _ = testTorrent1.Seeders[testSeeder1.ID]
	if foundSeeder != *testSeeder1 {
		t.Error("seeder not updated in local", testSeeder1)
	}
}

func TestSetLeecher(t *testing.T) {
	tx := createTestTx()
	testTorrent1 := createTestTorrent()
	panicErrNil(tx.AddTorrent(testTorrent1))
	testLeecher1 := createTestPeer(createTestUserID(), testTorrent1.ID)
	tx.AddLeecher(testTorrent1, testLeecher1)

	testLeecher1.Uploaded += 100

	tx.SetLeecher(testTorrent1, testLeecher1)
	foundTorrent, _, err := tx.FindTorrent(testTorrent1.Infohash)
	panicErrNil(err)
	foundLeecher, _ := foundTorrent.Leechers[testLeecher1.ID]
	if foundLeecher != *testLeecher1 {
		t.Error("leecher not updated in cache", testLeecher1)
	}
	foundLeecher, _ = testTorrent1.Leechers[testLeecher1.ID]
	if foundLeecher != *testLeecher1 {
		t.Error("leecher not updated in local", testLeecher1)
	}
}

func TestLeecherFinished(t *testing.T) {
	tx := createTestTx()
	testTorrent1 := createTestTorrent()
	panicErrNil(tx.AddTorrent(testTorrent1))
	testLeecher1 := createTestPeer(createTestUserID(), testTorrent1.ID)
	tx.AddLeecher(testTorrent1, testLeecher1)
	testLeecher1.Left = 0

	tx.LeecherFinished(testTorrent1, testLeecher1)
	foundTorrent, _, err := tx.FindTorrent(testTorrent1.Infohash)
	panicErrNil(err)
	foundSeeder, _ := foundTorrent.Seeders[testLeecher1.ID]
	if foundSeeder != *testLeecher1 {
		t.Error("seeder not added to cache", testLeecher1, foundSeeder)
	}
	foundSeeder, _ = foundTorrent.Leechers[testLeecher1.ID]
	if foundSeeder == *testLeecher1 {
		t.Error("leecher not removed from cache", testLeecher1)
	}
	foundSeeder, _ = testTorrent1.Seeders[testLeecher1.ID]
	if foundSeeder != *testLeecher1 {
		t.Error("seeder not added to local", testLeecher1)
	}
	foundSeeder, _ = testTorrent1.Leechers[testLeecher1.ID]
	if foundSeeder == *testLeecher1 {
		t.Error("leecher not removed from local", testLeecher1)
	}
}

// Add, update, verify remove
func TestUpdatePeer(t *testing.T) {
	tx := createTestTx()
	testTorrent1 := createTestTorrent()
	testSeeder1 := createTestPeer(createTestUserID(), testTorrent1.ID)
	panicErrNil(tx.AddTorrent(testTorrent1))
	panicErrNil(tx.AddSeeder(testTorrent1, testSeeder1))
	// Update a seeder, set it, then check to make sure it updated
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	testSeeder1.Uploaded += uint64(r.Int63())

	panicErrNil(tx.SetSeeder(testTorrent1, testSeeder1))

	panicErrNil(tx.RemoveSeeder(testTorrent1, testSeeder1))
	foundTorrent, _, err := tx.FindTorrent(testTorrent1.Infohash)
	panicErrNil(err)
	if seeder1, exists := foundTorrent.Seeders[testSeeder1.ID]; exists {
		t.Error("seeder not removed from cache", seeder1)
	}
	if seeder1, exists := testTorrent1.Seeders[testSeeder1.ID]; exists {
		t.Error("seeder not removed from local", seeder1)
	}
}
