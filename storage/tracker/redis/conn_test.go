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

  "github.com/pushrax/chihaya/config"
  "github.com/pushrax/chihaya/storage"
  "github.com/pushrax/chihaya/storage/tracker"
)

func createTestConn() tracker.Conn {
  testConfig, err := config.Open(os.Getenv("TESTCONFIGPATH"))
  panicOnErr(err)
  conf := &testConfig.Cache

  testPool, err := tracker.Open(conf)
  panicOnErr(err)

  newConn, err := testPool.Get()
  panicOnErr(err)

  return newConn
}

func TestFindUserSuccess(t *testing.T) {
  conn := createTestConn()
  testUser := createTestUser()

  panicOnErr(conn.AddUser(testUser))
  foundUser, found, err := conn.FindUser(testUser.Passkey)
  panicOnErr(err)
  if !found {
    t.Error("user not found", testUser)
  }
  if *foundUser != *testUser {
    t.Error("found user mismatch", *foundUser, testUser)
  }
  // Cleanup
  panicOnErr(conn.RemoveUser(testUser))
}

func TestFindUserFail(t *testing.T) {
  conn := createTestConn()
  testUser := createTestUser()

  foundUser, found, err := conn.FindUser(testUser.Passkey)
  panicOnErr(err)
  if found {
    t.Error("user found", foundUser)
  }
}

func TestRemoveUser(t *testing.T) {
  conn := createTestConn()
  testUser := createTestUser()

  panicOnErr(conn.AddUser(testUser))
  err := conn.RemoveUser(testUser)
  panicOnErr(err)
  foundUser, found, err := conn.FindUser(testUser.Passkey)
  panicOnErr(err)
  if found {
    t.Error("removed user found", foundUser)
  }
}

func TestFindTorrentSuccess(t *testing.T) {
  conn := createTestConn()
  testTorrent := createTestTorrent()
  panicOnErr(conn.AddTorrent(testTorrent))

  foundTorrent, found, err := conn.FindTorrent(testTorrent.Infohash)
  panicOnErr(err)
  if !found {
    t.Error("torrent not found", testTorrent)
  }
  if !reflect.DeepEqual(foundTorrent, testTorrent) {
    t.Error("found torrent mismatch", foundTorrent, testTorrent)
  }
  // Cleanup
  panicOnErr(conn.RemoveTorrent(testTorrent))
}

func TestFindTorrentFail(t *testing.T) {
  conn := createTestConn()
  testTorrent := createTestTorrent()

  foundTorrent, found, err := conn.FindTorrent(testTorrent.Infohash)
  panicOnErr(err)
  if found {
    t.Error("torrent found", foundTorrent)
  }
}

func TestRemoveTorrent(t *testing.T) {
  conn := createTestConn()
  testTorrent := createTestTorrent()
  panicOnErr(conn.AddTorrent(testTorrent))

  panicOnErr(conn.RemoveTorrent(testTorrent))
  foundTorrent, found, err := conn.FindTorrent(testTorrent.Infohash)
  panicOnErr(err)
  if found {
    t.Error("removed torrent found", foundTorrent)
  }
  // Cleanup
  panicOnErr(conn.RemoveTorrent(testTorrent))
}

func TestClientWhitelistSuccess(t *testing.T) {
  conn := createTestConn()
  testPeerID := "-lt0D30-"

  panicOnErr(conn.WhitelistClient(testPeerID))
  found, err := conn.ClientWhitelisted(testPeerID)
  panicOnErr(err)
  if !found {
    t.Error("peerID not found", testPeerID)
  }
  // Cleanup
  panicOnErr(conn.UnWhitelistClient(testPeerID))
}

func TestClientWhitelistFail(t *testing.T) {
  conn := createTestConn()
  testPeerID2 := "TIX0192"

  found, err := conn.ClientWhitelisted(testPeerID2)
  panicOnErr(err)
  if found {
    t.Error("peerID found", testPeerID2)
  }
}

func TestRecordSnatch(t *testing.T) {
  conn := createTestConn()
  testTorrent := createTestTorrent()
  testUser := createTestUser()
  panicOnErr(conn.AddTorrent(testTorrent))
  panicOnErr(conn.AddUser(testUser))

  userSnatches := testUser.Snatches
  torrentSnatches := testTorrent.Snatches

  panicOnErr(conn.RecordSnatch(testUser, testTorrent))

  foundTorrent, _, err := conn.FindTorrent(testTorrent.Infohash)
  panicOnErr(err)
  foundUser, _, err := conn.FindUser(testUser.Passkey)
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
  panicOnErr(conn.RemoveTorrent(testTorrent))
  panicOnErr(conn.RemoveUser(testUser))
}

func TestMarkActive(t *testing.T) {
  conn := createTestConn()
  testTorrent := createTestTorrent()
  testTorrent.Active = false
  panicOnErr(conn.AddTorrent(testTorrent))

  panicOnErr(conn.MarkActive(testTorrent))
  foundTorrent, _, err := conn.FindTorrent(testTorrent.Infohash)
  panicOnErr(err)

  if foundTorrent.Active != true {
    t.Error("cached torrent not activated")
  }
  if testTorrent.Active != true {
    t.Error("cached torrent not activated")
  }
  // Cleanup
  panicOnErr(conn.RemoveTorrent(testTorrent))
}

func TestClientWhitelistRemove(t *testing.T) {
  conn := createTestConn()
  testPeerID := "-lt0D30-"
  panicOnErr(conn.WhitelistClient(testPeerID))
  panicOnErr(conn.UnWhitelistClient(testPeerID))

  found, err := conn.ClientWhitelisted(testPeerID)
  panicOnErr(err)
  if found {
    t.Error("removed peerID found", testPeerID)
  }
}

func TestAddSeeder(t *testing.T) {
  conn := createTestConn()
  testTorrent := createTestTorrent()
  panicOnErr(conn.AddTorrent(testTorrent))
  testSeeder := createTestPeer(createTestUserID(), testTorrent.ID)

  panicOnErr(conn.AddSeeder(testTorrent, testSeeder))
  foundTorrent, found, err := conn.FindTorrent(testTorrent.Infohash)
  panicOnErr(err)
  foundSeeder, found := foundTorrent.Seeders[storage.PeerMapKey(testSeeder)]
  if found && foundSeeder != *testSeeder {
    t.Error("seeder not added to cache", testSeeder)
  }
  foundSeeder, found = testTorrent.Seeders[storage.PeerMapKey(testSeeder)]
  if found && foundSeeder != *testSeeder {
    t.Error("seeder not added to local", testSeeder)
  }
  // Cleanup
  panicOnErr(conn.RemoveTorrent(testTorrent))
}

func TestAddLeecher(t *testing.T) {
  conn := createTestConn()
  testTorrent := createTestTorrent()
  panicOnErr(conn.AddTorrent(testTorrent))
  testLeecher := createTestPeer(createTestUserID(), testTorrent.ID)

  panicOnErr(conn.AddLeecher(testTorrent, testLeecher))
  foundTorrent, found, err := conn.FindTorrent(testTorrent.Infohash)
  panicOnErr(err)
  foundLeecher, found := foundTorrent.Leechers[storage.PeerMapKey(testLeecher)]
  if found && foundLeecher != *testLeecher {
    t.Error("leecher not added to cache", testLeecher)
  }
  foundLeecher, found = testTorrent.Leechers[storage.PeerMapKey(testLeecher)]
  if found && foundLeecher != *testLeecher {
    t.Error("leecher not added to local", testLeecher)
  }
  // Cleanup
  panicOnErr(conn.RemoveTorrent(testTorrent))
}

func TestRemoveSeeder(t *testing.T) {
  conn := createTestConn()
  testTorrent := createTestTorrent()
  panicOnErr(conn.AddTorrent(testTorrent))
  testSeeder := createTestPeer(createTestUserID(), testTorrent.ID)
  panicOnErr(conn.AddSeeder(testTorrent, testSeeder))

  panicOnErr(conn.RemoveSeeder(testTorrent, testSeeder))
  foundSeeder, found := testTorrent.Seeders[storage.PeerMapKey(testSeeder)]
  if found || foundSeeder == *testSeeder {
    t.Error("seeder not removed from local", foundSeeder)
  }

  foundTorrent, found, err := conn.FindTorrent(testTorrent.Infohash)
  panicOnErr(err)
  foundSeeder, found = foundTorrent.Seeders[storage.PeerMapKey(testSeeder)]
  if found || foundSeeder == *testSeeder {
    t.Error("seeder not removed from cache", foundSeeder, *testSeeder)
  }
  // Cleanup
  panicOnErr(conn.RemoveTorrent(testTorrent))
}

func TestRemoveLeecher(t *testing.T) {
  conn := createTestConn()
  testTorrent := createTestTorrent()
  panicOnErr(conn.AddTorrent(testTorrent))
  testLeecher := createTestPeer(createTestUserID(), testTorrent.ID)
  panicOnErr(conn.AddLeecher(testTorrent, testLeecher))

  panicOnErr(conn.RemoveLeecher(testTorrent, testLeecher))
  foundTorrent, found, err := conn.FindTorrent(testTorrent.Infohash)
  panicOnErr(err)
  foundLeecher, found := foundTorrent.Leechers[storage.PeerMapKey(testLeecher)]
  if found || foundLeecher == *testLeecher {
    t.Error("leecher not removed from cache", foundLeecher, *testLeecher)
  }
  foundLeecher, found = testTorrent.Leechers[storage.PeerMapKey(testLeecher)]
  if found || foundLeecher == *testLeecher {
    t.Error("leecher not removed from local", foundLeecher, *testLeecher)
  }
  // Cleanup
  panicOnErr(conn.RemoveTorrent(testTorrent))
}

func TestSetSeeder(t *testing.T) {
  conn := createTestConn()
  testTorrent := createTestTorrent()
  panicOnErr(conn.AddTorrent(testTorrent))
  testSeeder := createTestPeer(createTestUserID(), testTorrent.ID)
  panicOnErr(conn.AddSeeder(testTorrent, testSeeder))

  r := rand.New(rand.NewSource(time.Now().UnixNano()))
  testSeeder.Uploaded += uint64(r.Int63())

  panicOnErr(conn.SetSeeder(testTorrent, testSeeder))

  foundTorrent, _, err := conn.FindTorrent(testTorrent.Infohash)
  panicOnErr(err)
  foundSeeder, _ := foundTorrent.Seeders[storage.PeerMapKey(testSeeder)]
  if foundSeeder != *testSeeder {
    t.Error("seeder not updated in cache", foundSeeder, *testSeeder)
  }
  foundSeeder, _ = testTorrent.Seeders[storage.PeerMapKey(testSeeder)]
  if foundSeeder != *testSeeder {
    t.Error("seeder not updated in local", foundSeeder, *testSeeder)
  }
  // Cleanup
  panicOnErr(conn.RemoveTorrent(testTorrent))
}

func TestSetLeecher(t *testing.T) {
  conn := createTestConn()
  testTorrent := createTestTorrent()
  panicOnErr(conn.AddTorrent(testTorrent))
  testLeecher := createTestPeer(createTestUserID(), testTorrent.ID)
  panicOnErr(conn.AddLeecher(testTorrent, testLeecher))

  r := rand.New(rand.NewSource(time.Now().UnixNano()))
  testLeecher.Uploaded += uint64(r.Int63())

  panicOnErr(conn.SetLeecher(testTorrent, testLeecher))
  foundTorrent, _, err := conn.FindTorrent(testTorrent.Infohash)
  panicOnErr(err)
  foundLeecher, _ := foundTorrent.Leechers[storage.PeerMapKey(testLeecher)]
  if foundLeecher != *testLeecher {
    t.Error("leecher not updated in cache", testLeecher)
  }
  foundLeecher, _ = testTorrent.Leechers[storage.PeerMapKey(testLeecher)]
  if foundLeecher != *testLeecher {
    t.Error("leecher not updated in local", testLeecher)
  }
  // Cleanup
  panicOnErr(conn.RemoveTorrent(testTorrent))
}

func TestIncrementSlots(t *testing.T) {
  conn := createTestConn()
  testUser := createTestUser()
  panicOnErr(conn.AddUser(testUser))
  numSlots := testUser.Slots

  panicOnErr(conn.IncrementSlots(testUser))
  foundUser, _, err := conn.FindUser(testUser.Passkey)
  panicOnErr(err)

  if foundUser.Slots != numSlots+1 {
    t.Error("cached slots not incremented")
  }
  if testUser.Slots != numSlots+1 {
    t.Error("local slots not incremented")
  }
  // Cleanup
  panicOnErr(conn.RemoveUser(testUser))
}

func TestDecrementSlots(t *testing.T) {
  conn := createTestConn()
  testUser := createTestUser()
  panicOnErr(conn.AddUser(testUser))
  numSlots := testUser.Slots

  panicOnErr(conn.DecrementSlots(testUser))
  foundUser, _, err := conn.FindUser(testUser.Passkey)
  panicOnErr(err)

  if foundUser.Slots != numSlots-1 {
    t.Error("cached slots not incremented")
  }
  if testUser.Slots != numSlots-1 {
    t.Error("local slots not incremented")
  }
  // Cleanup
  panicOnErr(conn.RemoveUser(testUser))
}

func TestLeecherFinished(t *testing.T) {
  conn := createTestConn()
  testTorrent := createTestTorrent()
  panicOnErr(conn.AddTorrent(testTorrent))
  testLeecher := createTestPeer(createTestUserID(), testTorrent.ID)
  panicOnErr(conn.AddLeecher(testTorrent, testLeecher))
  testLeecher.Left = 0

  panicOnErr(conn.LeecherFinished(testTorrent, testLeecher))

  foundTorrent, _, err := conn.FindTorrent(testTorrent.Infohash)
  panicOnErr(err)
  foundSeeder, _ := foundTorrent.Seeders[storage.PeerMapKey(testLeecher)]
  if foundSeeder != *testLeecher {
    t.Error("seeder not added to cache", foundSeeder, *testLeecher)
  }
  foundSeeder, _ = foundTorrent.Leechers[storage.PeerMapKey(testLeecher)]
  if foundSeeder == *testLeecher {
    t.Error("leecher not removed from cache", testLeecher)
  }
  foundSeeder, _ = testTorrent.Seeders[storage.PeerMapKey(testLeecher)]
  if foundSeeder != *testLeecher {
    t.Error("seeder not added to local", testLeecher)
  }
  foundSeeder, _ = testTorrent.Leechers[storage.PeerMapKey(testLeecher)]
  if foundSeeder == *testLeecher {
    t.Error("leecher not removed from local", testLeecher)
  }
  // Cleanup
  panicOnErr(conn.RemoveTorrent(testTorrent))
}

// Add, update, verify remove
func TestUpdatePeer(t *testing.T) {
  conn := createTestConn()
  testTorrent := createTestTorrent()
  testSeeder := createTestPeer(createTestUserID(), testTorrent.ID)
  panicOnErr(conn.AddTorrent(testTorrent))
  panicOnErr(conn.AddSeeder(testTorrent, testSeeder))
  // Update a seeder, set it, then check to make sure it updated
  r := rand.New(rand.NewSource(time.Now().UnixNano()))
  testSeeder.Uploaded += uint64(r.Int63())

  panicOnErr(conn.SetSeeder(testTorrent, testSeeder))

  panicOnErr(conn.RemoveSeeder(testTorrent, testSeeder))
  foundTorrent, _, err := conn.FindTorrent(testTorrent.Infohash)
  panicOnErr(err)
  if seeder, exists := foundTorrent.Seeders[storage.PeerMapKey(testSeeder)]; exists {
    t.Error("seeder not removed from cache", seeder)
  }
  if seeder, exists := testTorrent.Seeders[storage.PeerMapKey(testSeeder)]; exists {
    t.Error("seeder not removed from local", seeder)
  }
  // Cleanup
  panicOnErr(conn.RemoveTorrent(testTorrent))
}

func TestParallelFindUser(t *testing.T) {
  t.Parallel()
  if testing.Short() {
    t.Skip()
  }
  conn := createTestConn()
  testUserSuccess := createTestUser()
  testUserFail := createTestUser()
  panicOnErr(conn.AddUser(testUserSuccess))

  for i := 0; i < 10; i++ {
    foundUser, found, err := conn.FindUser(testUserFail.Passkey)
    panicOnErr(err)
    if found {
      t.Error("user found", foundUser)
    }
    foundUser, found, err = conn.FindUser(testUserSuccess.Passkey)
    panicOnErr(err)
    if !found {
      t.Error("user not found", testUserSuccess)
    }
    if *foundUser != *testUserSuccess {
      t.Error("found user mismatch", *foundUser, testUserSuccess)
    }
  }
  // Cleanup
  panicOnErr(conn.RemoveUser(testUserSuccess))
}

func TestParallelFindTorrent(t *testing.T) {
  t.Parallel()
  if testing.Short() {
    t.Skip()
  }
  conn := createTestConn()
  testTorrentSuccess := createTestTorrent()
  testTorrentFail := createTestTorrent()
  panicOnErr(conn.AddTorrent(testTorrentSuccess))

  for i := 0; i < 10; i++ {
    foundTorrent, found, err := conn.FindTorrent(testTorrentSuccess.Infohash)
    panicOnErr(err)
    if !found {
      t.Error("torrent not found", testTorrentSuccess)
    }
    if !reflect.DeepEqual(foundTorrent, testTorrentSuccess) {
      t.Error("found torrent mismatch", foundTorrent, testTorrentSuccess)
    }
    foundTorrent, found, err = conn.FindTorrent(testTorrentFail.Infohash)
    panicOnErr(err)
    if found {
      t.Error("torrent found", foundTorrent)
    }
  }
  // Cleanup
  panicOnErr(conn.RemoveTorrent(testTorrentSuccess))
}

func TestParallelSetSeeder(t *testing.T) {
  t.Parallel()
  if testing.Short() {
    t.Skip()
  }
  conn := createTestConn()
  testTorrent := createTestTorrent()
  panicOnErr(conn.AddTorrent(testTorrent))
  testSeeder := createTestPeer(createTestUserID(), testTorrent.ID)
  panicOnErr(conn.AddSeeder(testTorrent, testSeeder))
  r := rand.New(rand.NewSource(time.Now().UnixNano()))

  for i := 0; i < 10; i++ {
    testSeeder.Uploaded += uint64(r.Int63())

    panicOnErr(conn.SetSeeder(testTorrent, testSeeder))

    foundTorrent, _, err := conn.FindTorrent(testTorrent.Infohash)
    panicOnErr(err)
    foundSeeder, _ := foundTorrent.Seeders[storage.PeerMapKey(testSeeder)]
    if foundSeeder != *testSeeder {
      t.Error("seeder not updated in cache", foundSeeder, *testSeeder)
    }
    foundSeeder, _ = testTorrent.Seeders[storage.PeerMapKey(testSeeder)]
    if foundSeeder != *testSeeder {
      t.Error("seeder not updated in local", foundSeeder, *testSeeder)
    }
  }
  // Cleanup
  panicOnErr(conn.RemoveTorrent(testTorrent))
}

func TestParallelAddLeecher(t *testing.T) {
  t.Parallel()
  if testing.Short() {
    t.Skip()
  }
  conn := createTestConn()
  testTorrent := createTestTorrent()
  panicOnErr(conn.AddTorrent(testTorrent))

  for i := 0; i < 10; i++ {
    testLeecher := createTestPeer(createTestUserID(), testTorrent.ID)

    panicOnErr(conn.AddLeecher(testTorrent, testLeecher))

    foundTorrent, found, err := conn.FindTorrent(testTorrent.Infohash)
    panicOnErr(err)
    foundLeecher, found := foundTorrent.Leechers[storage.PeerMapKey(testLeecher)]
    if found && foundLeecher != *testLeecher {
      t.Error("leecher not added to cache", testLeecher)
    }
    foundLeecher, found = testTorrent.Leechers[storage.PeerMapKey(testLeecher)]
    if found && foundLeecher != *testLeecher {
      t.Error("leecher not added to local", testLeecher)
    }
  }
  // Cleanup
  panicOnErr(conn.RemoveTorrent(testTorrent))
}
