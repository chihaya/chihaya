// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package redis

import (
	"math/rand"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/pushrax/chihaya/cache"
	"github.com/pushrax/chihaya/config"
	"github.com/pushrax/chihaya/models"
)

func createTestTx() cache.Tx {
	testConfig, err := config.Open(os.Getenv("TESTCONFIGPATH"))
	panicOnErr(err)
	conf := &testConfig.Cache

	testPool, err := cache.Open(conf)
	panicOnErr(err)

	txObj, err := testPool.Get()
	panicOnErr(err)

	return txObj
}

func TestFindUserSuccess(t *testing.T) {
	tx := createTestTx()
	testUser := createTestUser()

	panicOnErr(tx.AddUser(testUser))
	foundUser, found, err := tx.FindUser(testUser.Passkey)
	panicOnErr(err)
	if !found {
		t.Error("user not found", testUser)
	}
	if *foundUser != *testUser {
		t.Error("found user mismatch", *foundUser, testUser)
	}
	// Cleanup
	panicOnErr(tx.RemoveUser(testUser))
}

func TestFindUserFail(t *testing.T) {
	tx := createTestTx()
	testUser := createTestUser()

	foundUser, found, err := tx.FindUser(testUser.Passkey)
	panicOnErr(err)
	if found {
		t.Error("user found", foundUser)
	}
}

func TestRemoveUser(t *testing.T) {
	tx := createTestTx()
	testUser := createTestUser()

	panicOnErr(tx.AddUser(testUser))
	err := tx.RemoveUser(testUser)
	panicOnErr(err)
	foundUser, found, err := tx.FindUser(testUser.Passkey)
	panicOnErr(err)
	if found {
		t.Error("removed user found", foundUser)
	}
}

func TestFindTorrentSuccess(t *testing.T) {
	tx := createTestTx()
	testTorrent := createTestTorrent()
	panicOnErr(tx.AddTorrent(testTorrent))

	foundTorrent, found, err := tx.FindTorrent(testTorrent.Infohash)
	panicOnErr(err)
	if !found {
		t.Error("torrent not found", testTorrent)
	}
	if !reflect.DeepEqual(foundTorrent, testTorrent) {
		t.Error("found torrent mismatch", foundTorrent, testTorrent)
	}
	// Cleanup
	panicOnErr(tx.RemoveTorrent(testTorrent))
}

func TestFindTorrentFail(t *testing.T) {
	tx := createTestTx()
	testTorrent := createTestTorrent()

	foundTorrent, found, err := tx.FindTorrent(testTorrent.Infohash)
	panicOnErr(err)
	if found {
		t.Error("torrent found", foundTorrent)
	}
}

func TestRemoveTorrent(t *testing.T) {
	tx := createTestTx()
	testTorrent := createTestTorrent()
	panicOnErr(tx.AddTorrent(testTorrent))

	panicOnErr(tx.RemoveTorrent(testTorrent))
	foundTorrent, found, err := tx.FindTorrent(testTorrent.Infohash)
	panicOnErr(err)
	if found {
		t.Error("removed torrent found", foundTorrent)
	}
	// Cleanup
	panicOnErr(tx.RemoveTorrent(testTorrent))
}

func TestClientWhitelistSuccess(t *testing.T) {
	tx := createTestTx()
	testPeerID := "-lt0D30-"

	panicOnErr(tx.WhitelistClient(testPeerID))
	found, err := tx.ClientWhitelisted(testPeerID)
	panicOnErr(err)
	if !found {
		t.Error("peerID not found", testPeerID)
	}
	// Cleanup
	panicOnErr(tx.UnWhitelistClient(testPeerID))
}

func TestClientWhitelistFail(t *testing.T) {
	tx := createTestTx()
	testPeerID2 := "TIX0192"

	found, err := tx.ClientWhitelisted(testPeerID2)
	panicOnErr(err)
	if found {
		t.Error("peerID found", testPeerID2)
	}
}

func TestRecordSnatch(t *testing.T) {
	tx := createTestTx()
	testTorrent := createTestTorrent()
	testUser := createTestUser()
	panicOnErr(tx.AddTorrent(testTorrent))
	panicOnErr(tx.AddUser(testUser))

	userSnatches := testUser.Snatches
	torrentSnatches := testTorrent.Snatches

	panicOnErr(tx.RecordSnatch(testUser, testTorrent))

	foundTorrent, _, err := tx.FindTorrent(testTorrent.Infohash)
	panicOnErr(err)
	foundUser, _, err := tx.FindUser(testUser.Passkey)
	panicOnErr(err)

	if testUser.Snatches != userSnatches+1 {
		t.Error("snatch not recorded to local user", testUser.Snatches, userSnatches+1)
	}
	if testTorrent.Snatches != torrentSnatches+1 {
		t.Error("snatch not recorded to local torrent")
	}
	if foundUser.Snatches != userSnatches+1 {
		t.Error("snatch not recorded to cached user", foundUser.Snatches, userSnatches+1)
	}
	if foundTorrent.Snatches != torrentSnatches+1 {
		t.Error("snatch not recorded to cached torrent")
	}
	// Cleanup
	panicOnErr(tx.RemoveTorrent(testTorrent))
	panicOnErr(tx.RemoveUser(testUser))
}

func TestMarkActive(t *testing.T) {
	tx := createTestTx()
	testTorrent := createTestTorrent()
	testTorrent.Active = false
	panicOnErr(tx.AddTorrent(testTorrent))

	panicOnErr(tx.MarkActive(testTorrent))
	foundTorrent, _, err := tx.FindTorrent(testTorrent.Infohash)
	panicOnErr(err)

	if foundTorrent.Active != true {
		t.Error("cached torrent not activated")
	}
	if testTorrent.Active != true {
		t.Error("cached torrent not activated")
	}
	// Cleanup
	panicOnErr(tx.RemoveTorrent(testTorrent))
}

func TestClientWhitelistRemove(t *testing.T) {
	tx := createTestTx()
	testPeerID := "-lt0D30-"
	panicOnErr(tx.WhitelistClient(testPeerID))
	panicOnErr(tx.UnWhitelistClient(testPeerID))

	found, err := tx.ClientWhitelisted(testPeerID)
	panicOnErr(err)
	if found {
		t.Error("removed peerID found", testPeerID)
	}
}

func TestAddSeeder(t *testing.T) {
	tx := createTestTx()
	testTorrent := createTestTorrent()
	panicOnErr(tx.AddTorrent(testTorrent))
	testSeeder := createTestPeer(createTestUserID(), testTorrent.ID)

	panicOnErr(tx.AddSeeder(testTorrent, testSeeder))
	foundTorrent, found, err := tx.FindTorrent(testTorrent.Infohash)
	panicOnErr(err)
	foundSeeder, found := foundTorrent.Seeders[models.PeerMapKey(testSeeder)]
	if found && foundSeeder != *testSeeder {
		t.Error("seeder not added to cache", testSeeder)
	}
	foundSeeder, found = testTorrent.Seeders[models.PeerMapKey(testSeeder)]
	if found && foundSeeder != *testSeeder {
		t.Error("seeder not added to local", testSeeder)
	}
	// Cleanup
	panicOnErr(tx.RemoveTorrent(testTorrent))
}

func TestAddLeecher(t *testing.T) {
	tx := createTestTx()
	testTorrent := createTestTorrent()
	panicOnErr(tx.AddTorrent(testTorrent))
	testLeecher := createTestPeer(createTestUserID(), testTorrent.ID)

	panicOnErr(tx.AddLeecher(testTorrent, testLeecher))
	foundTorrent, found, err := tx.FindTorrent(testTorrent.Infohash)
	panicOnErr(err)
	foundLeecher, found := foundTorrent.Leechers[models.PeerMapKey(testLeecher)]
	if found && foundLeecher != *testLeecher {
		t.Error("leecher not added to cache", testLeecher)
	}
	foundLeecher, found = testTorrent.Leechers[models.PeerMapKey(testLeecher)]
	if found && foundLeecher != *testLeecher {
		t.Error("leecher not added to local", testLeecher)
	}
	// Cleanup
	panicOnErr(tx.RemoveTorrent(testTorrent))
}

func TestRemoveSeeder(t *testing.T) {
	tx := createTestTx()
	testTorrent := createTestTorrent()
	panicOnErr(tx.AddTorrent(testTorrent))
	testSeeder := createTestPeer(createTestUserID(), testTorrent.ID)
	panicOnErr(tx.AddSeeder(testTorrent, testSeeder))

	panicOnErr(tx.RemoveSeeder(testTorrent, testSeeder))
	foundSeeder, found := testTorrent.Seeders[models.PeerMapKey(testSeeder)]
	if found || foundSeeder == *testSeeder {
		t.Error("seeder not removed from local", foundSeeder)
	}

	foundTorrent, found, err := tx.FindTorrent(testTorrent.Infohash)
	panicOnErr(err)
	foundSeeder, found = foundTorrent.Seeders[models.PeerMapKey(testSeeder)]
	if found || foundSeeder == *testSeeder {
		t.Error("seeder not removed from cache", foundSeeder, *testSeeder)
	}
	// Cleanup
	panicOnErr(tx.RemoveTorrent(testTorrent))
}

func TestRemoveLeecher(t *testing.T) {
	tx := createTestTx()
	testTorrent := createTestTorrent()
	panicOnErr(tx.AddTorrent(testTorrent))
	testLeecher := createTestPeer(createTestUserID(), testTorrent.ID)
	panicOnErr(tx.AddLeecher(testTorrent, testLeecher))

	panicOnErr(tx.RemoveLeecher(testTorrent, testLeecher))
	foundTorrent, found, err := tx.FindTorrent(testTorrent.Infohash)
	panicOnErr(err)
	foundLeecher, found := foundTorrent.Leechers[models.PeerMapKey(testLeecher)]
	if found || foundLeecher == *testLeecher {
		t.Error("leecher not removed from cache", foundLeecher, *testLeecher)
	}
	foundLeecher, found = testTorrent.Leechers[models.PeerMapKey(testLeecher)]
	if found || foundLeecher == *testLeecher {
		t.Error("leecher not removed from local", foundLeecher, *testLeecher)
	}
	// Cleanup
	panicOnErr(tx.RemoveTorrent(testTorrent))
}

func TestSetSeeder(t *testing.T) {
	tx := createTestTx()
	testTorrent := createTestTorrent()
	panicOnErr(tx.AddTorrent(testTorrent))
	testSeeder := createTestPeer(createTestUserID(), testTorrent.ID)
	panicOnErr(tx.AddSeeder(testTorrent, testSeeder))

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	testSeeder.Uploaded += uint64(r.Int63())

	panicOnErr(tx.SetSeeder(testTorrent, testSeeder))

	foundTorrent, _, err := tx.FindTorrent(testTorrent.Infohash)
	panicOnErr(err)
	foundSeeder, _ := foundTorrent.Seeders[models.PeerMapKey(testSeeder)]
	if foundSeeder != *testSeeder {
		t.Error("seeder not updated in cache", foundSeeder, *testSeeder)
	}
	foundSeeder, _ = testTorrent.Seeders[models.PeerMapKey(testSeeder)]
	if foundSeeder != *testSeeder {
		t.Error("seeder not updated in local", foundSeeder, *testSeeder)
	}
	// Cleanup
	panicOnErr(tx.RemoveTorrent(testTorrent))
}

func TestSetLeecher(t *testing.T) {
	tx := createTestTx()
	testTorrent := createTestTorrent()
	panicOnErr(tx.AddTorrent(testTorrent))
	testLeecher := createTestPeer(createTestUserID(), testTorrent.ID)
	panicOnErr(tx.AddLeecher(testTorrent, testLeecher))

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	testLeecher.Uploaded += uint64(r.Int63())

	panicOnErr(tx.SetLeecher(testTorrent, testLeecher))
	foundTorrent, _, err := tx.FindTorrent(testTorrent.Infohash)
	panicOnErr(err)
	foundLeecher, _ := foundTorrent.Leechers[models.PeerMapKey(testLeecher)]
	if foundLeecher != *testLeecher {
		t.Error("leecher not updated in cache", testLeecher)
	}
	foundLeecher, _ = testTorrent.Leechers[models.PeerMapKey(testLeecher)]
	if foundLeecher != *testLeecher {
		t.Error("leecher not updated in local", testLeecher)
	}
	// Cleanup
	panicOnErr(tx.RemoveTorrent(testTorrent))
}

func TestIncrementSlots(t *testing.T) {
	tx := createTestTx()
	testUser := createTestUser()
	panicOnErr(tx.AddUser(testUser))
	numSlots := testUser.Slots

	panicOnErr(tx.IncrementSlots(testUser))
	foundUser, _, err := tx.FindUser(testUser.Passkey)
	panicOnErr(err)

	if foundUser.Slots != numSlots+1 {
		t.Error("cached slots not incremented")
	}
	if testUser.Slots != numSlots+1 {
		t.Error("local slots not incremented")
	}
	// Cleanup
	panicOnErr(tx.RemoveUser(testUser))
}

func TestDecrementSlots(t *testing.T) {
	tx := createTestTx()
	testUser := createTestUser()
	panicOnErr(tx.AddUser(testUser))
	numSlots := testUser.Slots

	panicOnErr(tx.DecrementSlots(testUser))
	foundUser, _, err := tx.FindUser(testUser.Passkey)
	panicOnErr(err)

	if foundUser.Slots != numSlots-1 {
		t.Error("cached slots not incremented")
	}
	if testUser.Slots != numSlots-1 {
		t.Error("local slots not incremented")
	}
	// Cleanup
	panicOnErr(tx.RemoveUser(testUser))
}

func TestLeecherFinished(t *testing.T) {
	tx := createTestTx()
	testTorrent := createTestTorrent()
	panicOnErr(tx.AddTorrent(testTorrent))
	testLeecher := createTestPeer(createTestUserID(), testTorrent.ID)
	panicOnErr(tx.AddLeecher(testTorrent, testLeecher))
	testLeecher.Left = 0

	panicOnErr(tx.LeecherFinished(testTorrent, testLeecher))

	foundTorrent, _, err := tx.FindTorrent(testTorrent.Infohash)
	panicOnErr(err)
	foundSeeder, _ := foundTorrent.Seeders[models.PeerMapKey(testLeecher)]
	if foundSeeder != *testLeecher {
		t.Error("seeder not added to cache", foundSeeder, *testLeecher)
	}
	foundSeeder, _ = foundTorrent.Leechers[models.PeerMapKey(testLeecher)]
	if foundSeeder == *testLeecher {
		t.Error("leecher not removed from cache", testLeecher)
	}
	foundSeeder, _ = testTorrent.Seeders[models.PeerMapKey(testLeecher)]
	if foundSeeder != *testLeecher {
		t.Error("seeder not added to local", testLeecher)
	}
	foundSeeder, _ = testTorrent.Leechers[models.PeerMapKey(testLeecher)]
	if foundSeeder == *testLeecher {
		t.Error("leecher not removed from local", testLeecher)
	}
	// Cleanup
	panicOnErr(tx.RemoveTorrent(testTorrent))
}

// Add, update, verify remove
func TestUpdatePeer(t *testing.T) {
	tx := createTestTx()
	testTorrent := createTestTorrent()
	testSeeder := createTestPeer(createTestUserID(), testTorrent.ID)
	panicOnErr(tx.AddTorrent(testTorrent))
	panicOnErr(tx.AddSeeder(testTorrent, testSeeder))
	// Update a seeder, set it, then check to make sure it updated
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	testSeeder.Uploaded += uint64(r.Int63())

	panicOnErr(tx.SetSeeder(testTorrent, testSeeder))

	panicOnErr(tx.RemoveSeeder(testTorrent, testSeeder))
	foundTorrent, _, err := tx.FindTorrent(testTorrent.Infohash)
	panicOnErr(err)
	if seeder, exists := foundTorrent.Seeders[models.PeerMapKey(testSeeder)]; exists {
		t.Error("seeder not removed from cache", seeder)
	}
	if seeder, exists := testTorrent.Seeders[models.PeerMapKey(testSeeder)]; exists {
		t.Error("seeder not removed from local", seeder)
	}
	// Cleanup
	panicOnErr(tx.RemoveTorrent(testTorrent))
}

func TestParallelFindUser(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip()
	}
	tx := createTestTx()
	testUserSuccess := createTestUser()
	testUserFail := createTestUser()
	panicOnErr(tx.AddUser(testUserSuccess))

	for i := 0; i < 10; i++ {
		foundUser, found, err := tx.FindUser(testUserFail.Passkey)
		panicOnErr(err)
		if found {
			t.Error("user found", foundUser)
		}
		foundUser, found, err = tx.FindUser(testUserSuccess.Passkey)
		panicOnErr(err)
		if !found {
			t.Error("user not found", testUserSuccess)
		}
		if *foundUser != *testUserSuccess {
			t.Error("found user mismatch", *foundUser, testUserSuccess)
		}
	}
	// Cleanup
	panicOnErr(tx.RemoveUser(testUserSuccess))
}

func TestParallelFindTorrent(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip()
	}
	tx := createTestTx()
	testTorrentSuccess := createTestTorrent()
	testTorrentFail := createTestTorrent()
	panicOnErr(tx.AddTorrent(testTorrentSuccess))

	for i := 0; i < 10; i++ {
		foundTorrent, found, err := tx.FindTorrent(testTorrentSuccess.Infohash)
		panicOnErr(err)
		if !found {
			t.Error("torrent not found", testTorrentSuccess)
		}
		if !reflect.DeepEqual(foundTorrent, testTorrentSuccess) {
			t.Error("found torrent mismatch", foundTorrent, testTorrentSuccess)
		}
		foundTorrent, found, err = tx.FindTorrent(testTorrentFail.Infohash)
		panicOnErr(err)
		if found {
			t.Error("torrent found", foundTorrent)
		}
	}
	// Cleanup
	panicOnErr(tx.RemoveTorrent(testTorrentSuccess))
}

func TestParallelSetSeeder(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip()
	}
	tx := createTestTx()
	testTorrent := createTestTorrent()
	panicOnErr(tx.AddTorrent(testTorrent))
	testSeeder := createTestPeer(createTestUserID(), testTorrent.ID)
	panicOnErr(tx.AddSeeder(testTorrent, testSeeder))
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < 10; i++ {
		testSeeder.Uploaded += uint64(r.Int63())

		panicOnErr(tx.SetSeeder(testTorrent, testSeeder))

		foundTorrent, _, err := tx.FindTorrent(testTorrent.Infohash)
		panicOnErr(err)
		foundSeeder, _ := foundTorrent.Seeders[models.PeerMapKey(testSeeder)]
		if foundSeeder != *testSeeder {
			t.Error("seeder not updated in cache", foundSeeder, *testSeeder)
		}
		foundSeeder, _ = testTorrent.Seeders[models.PeerMapKey(testSeeder)]
		if foundSeeder != *testSeeder {
			t.Error("seeder not updated in local", foundSeeder, *testSeeder)
		}
	}
	// Cleanup
	panicOnErr(tx.RemoveTorrent(testTorrent))
}

func TestParallelAddLeecher(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip()
	}
	tx := createTestTx()
	testTorrent := createTestTorrent()
	panicOnErr(tx.AddTorrent(testTorrent))

	for i := 0; i < 10; i++ {
		testLeecher := createTestPeer(createTestUserID(), testTorrent.ID)

		panicOnErr(tx.AddLeecher(testTorrent, testLeecher))

		foundTorrent, found, err := tx.FindTorrent(testTorrent.Infohash)
		panicOnErr(err)
		foundLeecher, found := foundTorrent.Leechers[models.PeerMapKey(testLeecher)]
		if found && foundLeecher != *testLeecher {
			t.Error("leecher not added to cache", testLeecher)
		}
		foundLeecher, found = testTorrent.Leechers[models.PeerMapKey(testLeecher)]
		if found && foundLeecher != *testLeecher {
			t.Error("leecher not added to local", testLeecher)
		}
	}
	// Cleanup
	panicOnErr(tx.RemoveTorrent(testTorrent))
}
