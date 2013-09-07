// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package redis

import (
	"os"
	"testing"

	"github.com/garyburd/redigo/redis"

	"github.com/pushrax/chihaya/cache"
	"github.com/pushrax/chihaya/config"
	"github.com/pushrax/chihaya/models"
)

// Maximum number of parallel retries; depends on system latency
const MAX_RETRIES = 9000

const sample_infohash = "58c290f4ea1efb3adcb8c1ed2643232117577bcd"
const sample_passkey = "32426b162be0bce5428e7e36afaf734ae5afb355"

// Common interface for benchmarks and test error reporting
type TestReporter interface {
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Log(args ...interface{})
	Logf(format string, args ...interface{})
}

func verifyErrNil(err error, t TestReporter) {
	if err != nil {
		t.Error(err)
	}
}

// Legacy JSON support for benching
func (tx *Tx) initiateWrite() error {
	if tx.done {
		return cache.ErrTxDone
	}
	if tx.multi != true {
		tx.multi = true
		return tx.Send("MULTI")
	}
	return nil
}

func (tx *Tx) initiateRead() error {
	if tx.done {
		return cache.ErrTxDone
	}
	if tx.multi == true {
		panic("Tried to read during MULTI")
	}
	return nil
}

func createTestTxObj(t TestReporter) *Tx {
	testConfig, err := config.Open(os.Getenv("TESTCONFIGPATH"))
	conf := &testConfig.Cache
	verifyErrNil(err, t)

	testPool := &Pool{
		conf: conf,
		pool: redis.Pool{
			MaxIdle:      conf.MaxIdleConns,
			IdleTimeout:  conf.IdleTimeout.Duration,
			Dial:         makeDialFunc(conf),
			TestOnBorrow: testOnBorrow,
		},
	}

	//testDialFunc := makeDialFunc(&testConfig.Cache)
	//testConn, err := testDialFunc()
	txObj := &Tx{
		conf:  testPool.conf,
		done:  false,
		multi: false,
		Conn:  testPool.pool.Get(),
	}
	verifyErrNil(err, t)

	// Test connection before returning
	//txObj := Tx{&testConfig.Cache, false, false, testConn}
	_, err = txObj.Do("PING")
	verifyErrNil(err, t)
	return txObj
}

func createTestUser() models.User {
	testUser := models.User{214, "32426b162be0bce5428e7e36afaf734ae5afb355", 1.01, 1.0, 4, 2, 7}
	return testUser
}

func createSeeders() []models.Peer {
	testSeeders := make([]models.Peer, 4)
	testSeeders[0] = models.Peer{"testPeerID0", 57005, 48879, "testIP", 6889, 1024, 3000, 4200, 6}
	testSeeders[1] = models.Peer{"testPeerID1", 10101, 48879, "testIP", 6889, 1024, 3000, 4200, 6}
	testSeeders[2] = models.Peer{"testPeerID2", 29890, 48879, "testIP", 6889, 1024, 3000, 4200, 6}
	testSeeders[3] = models.Peer{"testPeerID3", 65261, 48879, "testIP", 6889, 1024, 3000, 4200, 6}
	return testSeeders
}

func createLeechers() []models.Peer {
	testLeechers := make([]models.Peer, 1)
	testLeechers[0] = models.Peer{"testPeerID", 11111, 48879, "testIP", 6889, 1024, 3000, 4200, 6}
	return testLeechers
}

func createTestTorrent() models.Torrent {

	testSeeders := createSeeders()
	testLeechers := createLeechers()

	seeders := make(map[string]models.Peer)
	for i := range testSeeders {
		seeders[testSeeders[i].ID] = testSeeders[i]
	}

	leechers := make(map[string]models.Peer)
	for i := range testLeechers {
		leechers[testLeechers[i].ID] = testLeechers[i]
	}

	testTorrent := models.Torrent{48879, sample_infohash, true, seeders, leechers, 11, 0.0, 0.0, 0}
	return testTorrent
}

func ExampleRedisTypeSchemaRemoveSeeder(torrent *models.Torrent, peer *models.Peer, t TestReporter) {
	testTx := createTestTxObj(t)
	setkey := testTx.conf.Prefix + "torrent:" + torrent.Infohash + ":seeders"
	reply, err := redis.Int(testTx.Do("SREM", setkey, *peer))
	if reply == 0 {
		t.Errorf("remove %v failed", *peer)
	}
	verifyErrNil(err, t)
}

func ExampleRedisTypesSchemaFindUser(passkey string, t TestReporter) (*models.User, bool) {
	testTx := createTestTxObj(t)
	hashkey := testTx.conf.Prefix + UserPrefix + passkey
	userVals, err := redis.Strings(testTx.Do("HVALS", hashkey))
	if len(userVals) == 0 {
		return nil, false
	}
	verifyErrNil(err, t)
	compareUser, err := createUser(userVals)
	verifyErrNil(err, t)
	return compareUser, true
}

func TestFindUserSuccess(t *testing.T) {
	testUser := createTestUser()
	testTx := createTestTxObj(t)
	hashkey := testTx.conf.Prefix + UserPrefix + sample_passkey
	_, err := testTx.Do("DEL", hashkey)
	verifyErrNil(err, t)

	err = testTx.AddUser(&testUser)
	verifyErrNil(err, t)

	compareUser, exists := ExampleRedisTypesSchemaFindUser(sample_passkey, t)

	if !exists {
		t.Error("User not found!")
	}
	if testUser != *compareUser {
		t.Errorf("user mismatch: %v vs. %v", compareUser, testUser)
	}
}

func TestFindUserFail(t *testing.T) {
	compareUser, exists := ExampleRedisTypesSchemaFindUser("not_a_user_passkey", t)
	if exists {
		t.Errorf("User %v found when none should exist!", compareUser)
	}
}

func TestAddGetPeers(t *testing.T) {

	testTx := createTestTxObj(t)
	testTorrent := createTestTorrent()

	setkey := testTx.conf.Prefix + "torrent:" + testTorrent.Infohash + ":seeders"
	testTx.Do("DEL", setkey)

	testTx.addPeers(testTorrent.Infohash, testTorrent.Seeders, ":seeders")
	peerMap, err := testTx.getPeers(sample_infohash, ":seeders")
	if err != nil {
		t.Error(err)
	} else if len(peerMap) != len(testTorrent.Seeders) {
		t.Error("Num Peers not equal")
	}
}

func BenchmarkRedisTypesSchemaRemoveSeeder(b *testing.B) {
	for bCount := 0; bCount < b.N; bCount++ {
		// Ensure that remove completes successfully,
		// even if it doesn't impact the performance
		b.StopTimer()
		testTx := createTestTxObj(b)
		testTorrent := createTestTorrent()
		setkey := testTx.conf.Prefix + "torrent:" + testTorrent.Infohash + ":seeders"
		testSeeders := createSeeders()
		reply, err := redis.Int(testTx.Do("SADD", setkey,
			testSeeders[0],
			testSeeders[1],
			testSeeders[2],
			testSeeders[3]))

		if reply == 0 {
			b.Log("no keys added!")
		}
		verifyErrNil(err, b)
		b.StartTimer()

		ExampleRedisTypeSchemaRemoveSeeder(&testTorrent, &testSeeders[2], b)
	}
}

func BenchmarkRedisTypesSchemaFindUser(b *testing.B) {

	// Ensure successful user find ( a failed lookup may have different performance )
	b.StopTimer()
	testUser := createTestUser()
	testTx := createTestTxObj(b)
	hashkey := testTx.conf.Prefix + UserPrefix + sample_passkey
	reply, err := testTx.Do("HMSET", hashkey,
		"id", testUser.ID,
		"passkey", testUser.Passkey,
		"up_multiplier", testUser.UpMultiplier,
		"down_multiplier", testUser.DownMultiplier,
		"slots", testUser.Slots,
		"slots_used", testUser.SlotsUsed)

	if reply == nil {
		b.Log("no hash fields added!")
	}
	verifyErrNil(err, b)
	b.StartTimer()

	for bCount := 0; bCount < b.N; bCount++ {

		compareUser, exists := ExampleRedisTypesSchemaFindUser(sample_passkey, b)

		b.StopTimer()
		if !exists {
			b.Error("User not found!")
		}
		if testUser != *compareUser {
			b.Errorf("user mismatch: %v vs. %v", compareUser, testUser)
		}
		b.StartTimer()
	}
}

func TestReadAfterWrite(t *testing.T) {
	// Test requires panic
	defer func() {
		if err := recover(); err == nil {
			t.Error("Read after write did not panic")
		}
	}()

	testTx := createTestTxObj(t)
	verifyErrNil(testTx.initiateWrite(), t)
	verifyErrNil(testTx.initiateRead(), t)
}

func TestCloseClosedTransaction(t *testing.T) {
	//require panic
	defer func() {
		if err := recover(); err == nil {
			t.Error("Closing a closed transaction did not panic")
		}
	}()

	testTx := createTestTxObj(t)
	testTx.close()
	testTx.close()
}
