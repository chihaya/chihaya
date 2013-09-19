// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package redis

import (
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/pushrax/chihaya/cache"
	"github.com/pushrax/chihaya/config"
	"github.com/pushrax/chihaya/models"
)

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
	testUser := createTestUser()

	panicErrNil(tx.AddUser(testUser))
	foundUser, found, err := tx.FindUser(testUser.Passkey)
	panicErrNil(err)
	if !found {
		t.Error("user not found", testUser)
	}
	if *foundUser != *testUser {
		t.Error("found user mismatch", *foundUser, testUser)
	}
}

func TestFindUserFail(t *testing.T) {
	tx := createTestTx()
	testUser := createTestUser()

	foundUser, found, err := tx.FindUser(testUser.Passkey)
	panicErrNil(err)
	if found {
		t.Error("user found", foundUser)
	}
}

func TestRemoveUser(t *testing.T) {
	tx := createTestTx()
	testUser := createTestUser()

	panicErrNil(tx.AddUser(testUser))
	err := tx.RemoveUser(testUser)
	panicErrNil(err)
	foundUser, found, err := tx.FindUser(testUser.Passkey)
	panicErrNil(err)
	if found {
		t.Error("removed user found", foundUser)
	}
}

func TestFindTorrentSuccess(t *testing.T) {
	tx := createTestTx()
	testTorrent := createTestTorrent()
	panicErrNil(tx.AddTorrent(testTorrent))

	foundTorrent, found, err := tx.FindTorrent(testTorrent.Infohash)
	panicErrNil(err)
	if !found {
		t.Error("torrent not found", testTorrent)
	}
	if !torrentsEqual(foundTorrent, testTorrent) {
		t.Error("found torrent mismatch", foundTorrent, testTorrent)
	}
}

func TestFindTorrentFail(t *testing.T) {
	tx := createTestTx()
	testTorrent := createTestTorrent()

	foundTorrent, found, err := tx.FindTorrent(testTorrent.Infohash)
	panicErrNil(err)
	if found {
		t.Error("torrent found", foundTorrent)
	}
}

func TestRemoveTorrent(t *testing.T) {
	tx := createTestTx()
	testTorrent := createTestTorrent()
	panicErrNil(tx.AddTorrent(testTorrent))

	panicErrNil(tx.RemoveTorrent(testTorrent))
	foundTorrent, found, err := tx.FindTorrent(testTorrent.Infohash)
	panicErrNil(err)
	if found {
		t.Error("removed torrent found", foundTorrent)
	}
}

func TestClientWhitelistSuccess(t *testing.T) {
	tx := createTestTx()
	testPeerID := "-lt0D30-"

	panicErrNil(tx.WhitelistClient(testPeerID))
	found, err := tx.ClientWhitelisted(testPeerID)
	panicErrNil(err)
	if !found {
		t.Error("peerID not found", testPeerID)
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

func TestRecordSnatch(t *testing.T) {
	tx := createTestTx()
	testTorrent := createTestTorrent()
	testUser := createTestUser()
	panicErrNil(tx.AddTorrent(testTorrent))
	panicErrNil(tx.AddUser(testUser))

	userSnatches := testUser.Snatches
	torrentSnatches := testTorrent.Snatches

	panicErrNil(tx.RecordSnatch(testUser, testTorrent))

	foundTorrent, _, err := tx.FindTorrent(testTorrent.Infohash)
	panicErrNil(err)
	foundUser, _, err := tx.FindUser(testUser.Passkey)
	panicErrNil(err)

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
}

func TestMarkActive(t *testing.T) {
	tx := createTestTx()
	testTorrent := createTestTorrent()
	testTorrent.Active = false
	panicErrNil(tx.AddTorrent(testTorrent))

	panicErrNil(tx.MarkActive(testTorrent))
	foundTorrent, _, err := tx.FindTorrent(testTorrent.Infohash)
	panicErrNil(err)

	if foundTorrent.Active != true {
		t.Error("cached torrent not activated")
	}
	if testTorrent.Active != true {
		t.Error("cached torrent not activated")
	}
}

func TestClientWhitelistRemove(t *testing.T) {
	tx := createTestTx()
	testPeerID := "-lt0D30-"
	panicErrNil(tx.WhitelistClient(testPeerID))
	panicErrNil(tx.UnWhitelistClient(testPeerID))

	found, err := tx.ClientWhitelisted(testPeerID)
	panicErrNil(err)
	if found {
		t.Error("removed peerID found", testPeerID)
	}
}

func TestAddSeeder(t *testing.T) {
	tx := createTestTx()
	testTorrent := createTestTorrent()
	panicErrNil(tx.AddTorrent(testTorrent))
	testSeeder := createTestPeer(createTestUserID(), testTorrent.ID)

	panicErrNil(tx.AddSeeder(testTorrent, testSeeder))
	foundTorrent, found, err := tx.FindTorrent(testTorrent.Infohash)
	panicErrNil(err)
	foundSeeder, found := foundTorrent.Seeders[models.PeerMapKey(testSeeder)]
	if found && foundSeeder != *testSeeder {
		t.Error("seeder not added to cache", testSeeder)
	}
	foundSeeder, found = testTorrent.Seeders[models.PeerMapKey(testSeeder)]
	if found && foundSeeder != *testSeeder {
		t.Error("seeder not added to local", testSeeder)
	}
}

func TestAddLeecher(t *testing.T) {
	tx := createTestTx()
	testTorrent := createTestTorrent()
	panicErrNil(tx.AddTorrent(testTorrent))
	testLeecher := createTestPeer(createTestUserID(), testTorrent.ID)

	panicErrNil(tx.AddLeecher(testTorrent, testLeecher))
	foundTorrent, found, err := tx.FindTorrent(testTorrent.Infohash)
	panicErrNil(err)
	foundLeecher, found := foundTorrent.Leechers[models.PeerMapKey(testLeecher)]
	if found && foundLeecher != *testLeecher {
		t.Error("leecher not added to cache", testLeecher)
	}
	foundLeecher, found = testTorrent.Leechers[models.PeerMapKey(testLeecher)]
	if found && foundLeecher != *testLeecher {
		t.Error("leecher not added to local", testLeecher)
	}
}

func TestRemoveSeeder(t *testing.T) {
	tx := createTestTx()
	testTorrent := createTestTorrent()
	panicErrNil(tx.AddTorrent(testTorrent))
	testSeeder := createTestPeer(createTestUserID(), testTorrent.ID)
	panicErrNil(tx.AddSeeder(testTorrent, testSeeder))

	panicErrNil(tx.RemoveSeeder(testTorrent, testSeeder))
	foundSeeder, found := testTorrent.Seeders[models.PeerMapKey(testSeeder)]
	if found || foundSeeder == *testSeeder {
		t.Error("seeder not removed from local", foundSeeder)
	}

	foundTorrent, found, err := tx.FindTorrent(testTorrent.Infohash)
	panicErrNil(err)
	foundSeeder, found = foundTorrent.Seeders[models.PeerMapKey(testSeeder)]
	if found || foundSeeder == *testSeeder {
		t.Error("seeder not removed from cache", foundSeeder, *testSeeder)
	}
}

func TestRemoveLeecher(t *testing.T) {
	tx := createTestTx()
	testTorrent := createTestTorrent()
	panicErrNil(tx.AddTorrent(testTorrent))
	testLeecher := createTestPeer(createTestUserID(), testTorrent.ID)
	panicErrNil(tx.AddLeecher(testTorrent, testLeecher))

	panicErrNil(tx.RemoveLeecher(testTorrent, testLeecher))
	foundTorrent, found, err := tx.FindTorrent(testTorrent.Infohash)
	panicErrNil(err)
	foundLeecher, found := foundTorrent.Leechers[models.PeerMapKey(testLeecher)]
	if found || foundLeecher == *testLeecher {
		t.Error("leecher not removed from cache", foundLeecher, *testLeecher)
	}
	foundLeecher, found = testTorrent.Leechers[models.PeerMapKey(testLeecher)]
	if found || foundLeecher == *testLeecher {
		t.Error("leecher not removed from local", foundLeecher, *testLeecher)
	}
}

func TestSetSeeder(t *testing.T) {
	tx := createTestTx()
	testTorrent := createTestTorrent()
	panicErrNil(tx.AddTorrent(testTorrent))
	testSeeder := createTestPeer(createTestUserID(), testTorrent.ID)
	panicErrNil(tx.AddSeeder(testTorrent, testSeeder))

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	testSeeder.Uploaded += uint64(r.Int63())

	panicErrNil(tx.SetSeeder(testTorrent, testSeeder))

	foundTorrent, _, err := tx.FindTorrent(testTorrent.Infohash)
	panicErrNil(err)
	foundSeeder, _ := foundTorrent.Seeders[models.PeerMapKey(testSeeder)]
	if foundSeeder != *testSeeder {
		t.Error("seeder not updated in cache", foundSeeder, *testSeeder)
	}
	foundSeeder, _ = testTorrent.Seeders[models.PeerMapKey(testSeeder)]
	if foundSeeder != *testSeeder {
		t.Error("seeder not updated in local", foundSeeder, *testSeeder)
	}
}

func TestSetLeecher(t *testing.T) {
	tx := createTestTx()
	testTorrent := createTestTorrent()
	panicErrNil(tx.AddTorrent(testTorrent))
	testLeecher := createTestPeer(createTestUserID(), testTorrent.ID)
	panicErrNil(tx.AddLeecher(testTorrent, testLeecher))

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	testLeecher.Uploaded += uint64(r.Int63())

	panicErrNil(tx.SetLeecher(testTorrent, testLeecher))
	foundTorrent, _, err := tx.FindTorrent(testTorrent.Infohash)
	panicErrNil(err)
	foundLeecher, _ := foundTorrent.Leechers[models.PeerMapKey(testLeecher)]
	if foundLeecher != *testLeecher {
		t.Error("leecher not updated in cache", testLeecher)
	}
	foundLeecher, _ = testTorrent.Leechers[models.PeerMapKey(testLeecher)]
	if foundLeecher != *testLeecher {
		t.Error("leecher not updated in local", testLeecher)
	}
}

func TestIncrementSlots(t *testing.T) {
	tx := createTestTx()
	testUser := createTestUser()
	panicErrNil(tx.AddUser(testUser))
	numSlots := testUser.Slots

	panicErrNil(tx.IncrementSlots(testUser))
	foundUser, _, err := tx.FindUser(testUser.Passkey)
	panicErrNil(err)

	if foundUser.Slots != numSlots+1 {
		t.Error("cached slots not incremented")
	}
	if testUser.Slots != numSlots+1 {
		t.Error("local slots not incremented")
	}
}

func TestDecrementSlots(t *testing.T) {
	tx := createTestTx()
	testUser := createTestUser()
	panicErrNil(tx.AddUser(testUser))
	numSlots := testUser.Slots

	panicErrNil(tx.DecrementSlots(testUser))
	foundUser, _, err := tx.FindUser(testUser.Passkey)
	panicErrNil(err)

	if foundUser.Slots != numSlots-1 {
		t.Error("cached slots not incremented")
	}
	if testUser.Slots != numSlots-1 {
		t.Error("local slots not incremented")
	}
}

func TestLeecherFinished(t *testing.T) {
	tx := createTestTx()
	testTorrent := createTestTorrent()
	panicErrNil(tx.AddTorrent(testTorrent))
	testLeecher := createTestPeer(createTestUserID(), testTorrent.ID)
	panicErrNil(tx.AddLeecher(testTorrent, testLeecher))
	testLeecher.Left = 0

	panicErrNil(tx.LeecherFinished(testTorrent, testLeecher))

	foundTorrent, _, err := tx.FindTorrent(testTorrent.Infohash)
	panicErrNil(err)
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
}

// Add, update, verify remove
func TestUpdatePeer(t *testing.T) {
	tx := createTestTx()
	testTorrent := createTestTorrent()
	testSeeder := createTestPeer(createTestUserID(), testTorrent.ID)
	panicErrNil(tx.AddTorrent(testTorrent))
	panicErrNil(tx.AddSeeder(testTorrent, testSeeder))
	// Update a seeder, set it, then check to make sure it updated
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	testSeeder.Uploaded += uint64(r.Int63())

	panicErrNil(tx.SetSeeder(testTorrent, testSeeder))

	panicErrNil(tx.RemoveSeeder(testTorrent, testSeeder))
	foundTorrent, _, err := tx.FindTorrent(testTorrent.Infohash)
	panicErrNil(err)
	if seeder, exists := foundTorrent.Seeders[models.PeerMapKey(testSeeder)]; exists {
		t.Error("seeder not removed from cache", seeder)
	}
	if seeder, exists := testTorrent.Seeders[models.PeerMapKey(testSeeder)]; exists {
		t.Error("seeder not removed from local", seeder)
	}
}

func TestParallelFindUser(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip()
	}
	tx := createTestTx()
	testUserSuccess := createTestUser()
	testUserFail := createTestUser()
	panicErrNil(tx.AddUser(testUserSuccess))

	for i := 0; i < 10; i++ {
		foundUser, found, err := tx.FindUser(testUserFail.Passkey)
		panicErrNil(err)
		if found {
			t.Error("user found", foundUser)
		}
		foundUser, found, err = tx.FindUser(testUserSuccess.Passkey)
		panicErrNil(err)
		if !found {
			t.Error("user not found", testUserSuccess)
		}
		if *foundUser != *testUserSuccess {
			t.Error("found user mismatch", *foundUser, testUserSuccess)
		}
	}
}

func TestParallelFindTorrent(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip()
	}
	tx := createTestTx()
	testTorrentSuccess := createTestTorrent()
	testTorrentFail := createTestTorrent()
	panicErrNil(tx.AddTorrent(testTorrentSuccess))

	for i := 0; i < 10; i++ {
		foundTorrent, found, err := tx.FindTorrent(testTorrentSuccess.Infohash)
		panicErrNil(err)
		if !found {
			t.Error("torrent not found", testTorrentSuccess)
		}
		if !torrentsEqual(foundTorrent, testTorrentSuccess) {
			t.Error("found torrent mismatch", foundTorrent, testTorrentSuccess)
		}
		foundTorrent, found, err = tx.FindTorrent(testTorrentFail.Infohash)
		panicErrNil(err)
		if found {
			t.Error("torrent found", foundTorrent)
		}
	}
}

func TestParallelSetSeeder(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip()
	}
	tx := createTestTx()
	testTorrent := createTestTorrent()
	panicErrNil(tx.AddTorrent(testTorrent))
	testSeeder := createTestPeer(createTestUserID(), testTorrent.ID)
	panicErrNil(tx.AddSeeder(testTorrent, testSeeder))
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < 10; i++ {
		testSeeder.Uploaded += uint64(r.Int63())

		panicErrNil(tx.SetSeeder(testTorrent, testSeeder))

		foundTorrent, _, err := tx.FindTorrent(testTorrent.Infohash)
		panicErrNil(err)
		foundSeeder, _ := foundTorrent.Seeders[models.PeerMapKey(testSeeder)]
		if foundSeeder != *testSeeder {
			t.Error("seeder not updated in cache", foundSeeder, *testSeeder)
		}
		foundSeeder, _ = testTorrent.Seeders[models.PeerMapKey(testSeeder)]
		if foundSeeder != *testSeeder {
			t.Error("seeder not updated in local", foundSeeder, *testSeeder)
		}
	}
}

func TestParallelAddLeecher(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip()
	}
	tx := createTestTx()
	testTorrent := createTestTorrent()
	panicErrNil(tx.AddTorrent(testTorrent))

	for i := 0; i < 10; i++ {
		testLeecher := createTestPeer(createTestUserID(), testTorrent.ID)

		panicErrNil(tx.AddLeecher(testTorrent, testLeecher))

		foundTorrent, found, err := tx.FindTorrent(testTorrent.Infohash)
		panicErrNil(err)
		foundLeecher, found := foundTorrent.Leechers[models.PeerMapKey(testLeecher)]
		if found && foundLeecher != *testLeecher {
			t.Error("leecher not added to cache", testLeecher)
		}
		foundLeecher, found = testTorrent.Leechers[models.PeerMapKey(testLeecher)]
		if found && foundLeecher != *testLeecher {
			t.Error("leecher not added to local", testLeecher)
		}
	}
}
