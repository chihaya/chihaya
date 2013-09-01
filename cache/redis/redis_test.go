// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package redis implements the storage interface for a BitTorrent tracker.
// Benchmarks are at the top of the file, tests are at the bottom
package redis

import (
	"encoding/json"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/garyburd/redigo/redis"

	"github.com/pushrax/chihaya/cache"
	"github.com/pushrax/chihaya/config"
	"github.com/pushrax/chihaya/models"
)

// Maximum number of parallel retries, will depends on system latency
const MAX_RETRIES = 9000

type TestReporter interface {
	Error(args ...interface{})
}

func verifyErrNil(err error, t TestReporter) {
	if err != nil {
		t.Error(err)
	}
}

func createTestTxObj(t *testing.T) *Tx {
	testConfig, err := config.Open(os.Getenv("TESTCONFIGPATH"))
	conf := &testConfig.Cache
	verifyErrNil(err, t)

	testPool := &Pool{
		conf: conf,
		pool: redis.Pool{
			MaxIdle:      conf.MaxIdleConn,
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

func sampleTransaction(testTx *Tx, retries int, t *testing.T) {
	defer func() {
		if rawError := recover(); rawError != nil {
			t.Error(rawError)
		}
	}()
	verifyErrNil(testTx.initiateRead(), t)
	_, err := testTx.Do("WATCH", "testKeyA")
	verifyErrNil(err, t)

	_, err = redis.String(testTx.Do("GET", "testKeyA"))
	if err != nil {
		if err == redis.ErrNil {
			t.Log("testKeyA does not exist yet")
		} else {
			t.Error(err)
		}
	}
	_, err = testTx.Do("WATCH", "testKeyB")
	if err != nil {
		if err == redis.ErrNil {
			t.Log("redis.ErrNil")
		} else {
			t.Error(err)
		}
	}

	_, err = redis.String(testTx.Do("GET", "testKeyB"))
	if err != nil {
		if err == redis.ErrNil {
			t.Log("testKeyB does not exist yet")
		} else {
			t.Error(err)
		}
	}

	verifyErrNil(testTx.initiateWrite(), t)

	// Generate random data to set
	randGen := rand.New(rand.NewSource(time.Now().UnixNano()))
	verifyErrNil(testTx.Send("SET", "testKeyA", strconv.Itoa(randGen.Int())), t)
	verifyErrNil(testTx.Send("SET", "testKeyB", strconv.Itoa(randGen.Int())), t)

	err = testTx.Commit()
	// For parallel runs, there may be conflicts, retry until successful
	if err == cache.ErrTxConflict && retries > 0 {
		sampleTransaction(testTx, retries-1, t)
		// Clear TxConflict, if retries max out, errors are already recorded
		err = nil
	} else if err == cache.ErrTxConflict {
		t.Error("Conflict encountered, max retries reached")
		t.Error(err)
	}
	verifyErrNil(err, t)
}

func BenchmarkRedisJsonSchemaRemoveSeeder(b *testing.B) {
	var t testing.T
	infohash := "58c290f4ea1efb3adcb8c1ed2643232117577bcd"
	for bCount := 0; bCount < b.N; bCount++ {
		testTx := createTestTxObj(&t)

		verifyErrNil(testTx.initiateRead(), b)

		key := testTx.conf.Prefix + "torrent:" + infohash
		_, err := testTx.Do("WATCH", key)
		reply, err := redis.String(testTx.Do("GET", key))
		if err != nil {
			if err == redis.ErrNil {
				b.Logf("no key yet, creating json reply")
				reply = string(createTorrentJson(&t))
			} else {
				b.Error(err)
			}
		}

		torrent := &models.Torrent{}
		verifyErrNil(json.NewDecoder(strings.NewReader(reply)).Decode(torrent), b)

		delete(torrent.Seeders, "testPeerID2")

		jsonTorrent, err := json.Marshal(torrent)
		verifyErrNil(err, b)
		verifyErrNil(testTx.initiateWrite(), b)
		verifyErrNil(testTx.Send("SET", key, jsonTorrent), b)
		verifyErrNil(testTx.Commit(), b)
	}
}

func BenchmarkRedisTypesSchemaRemoveSeeder(b *testing.B) {
	var t testing.T
	infohash := "58c290f4ea1efb3adcb8c1ed2643232117577bcd"
	for bCount := 0; bCount < b.N; bCount++ {
		testTx := createTestTxObj(&t)
		setkey := testTx.conf.Prefix + "torrent:" + infohash + ":seeders"

		b.StopTimer()
		reply, err := redis.Int(testTx.Do("SADD", setkey, "testPeerID0", "testPeerID1", "testPeerID2", "testPeerID3"))
		if reply == 0 {
			b.Error("no keys added!")
		}
		verifyErrNil(err, b)

		b.StartTimer()

		reply, err = redis.Int(testTx.Do("SREM", setkey, "testPeerID2"))
		if reply == 0 {
			b.Error("remove failed")
		}
		verifyErrNil(err, b)
	}

}

func BenchmarkRedisJsonSchemaFindUser(b *testing.B) {
	var t testing.T
	passkey := "32426b162be0bce5428e7e36afaf734ae5afb355"
	for bCount := 0; bCount < b.N; bCount++ {
		testTx := createTestTxObj(&t)

		verifyErrNil(testTx.initiateRead(), b)

		key := testTx.conf.Prefix + "user:" + passkey
		_, err := testTx.Do("WATCH", key)
		verifyErrNil(err, b)
		reply, err := redis.String(testTx.Do("GET", key))
		if err != nil {
			if err == redis.ErrNil {
				b.Logf("no key yet, creating & sending json back to redis")
				reply = string(createUserJson(&t))
				// Add user to redis, as this is a read-only operation
				verifyErrNil(testTx.initiateWrite(), b)
				verifyErrNil(testTx.Send("SET", key, reply), b)
				verifyErrNil(testTx.Commit(), b)
			} else {
				b.Error(err)
			}
		}

		user := &models.User{}
		verifyErrNil(json.NewDecoder(strings.NewReader(reply)).Decode(user), b)
	}
}

func BenchmarkRedisTypesSchemaFindUser(b *testing.B) {
	var t testing.T
	passkey := "32426b162be0bce5428e7e36afaf734ae5afb355"
	for bCount := 0; bCount < b.N; bCount++ {
		testTx := createTestTxObj(&t)
		hashkey := testTx.conf.Prefix + "user_hash:" + passkey

		b.StopTimer()
		testUser := createUser()
		reply, err := testTx.Do("HMSET", hashkey, "id", testUser.ID, "passkey", testUser.Passkey, "up_multiplier", testUser.UpMultiplier, "down_multiplier", testUser.DownMultiplier, "slots", testUser.Slots, "slots_used", testUser.SlotsUsed)
		if reply == nil {
			b.Error("no hash fields added!")
		}
		verifyErrNil(err, b)

		b.StartTimer()

		userVals, err := redis.Strings(testTx.Do("HVALS", hashkey))
		if userVals == nil {
			b.Error("user does not exist!")
		}
		verifyErrNil(err, b)
		compareUser := createUserFromValues(userVals, &t)
		if testUser != compareUser {
			b.Errorf("user mismatch: %v vs. %v", compareUser, testUser)
			b.Log(userVals)
		}
	}
}

func createTorrentJson(t *testing.T) []byte {
	jsonTorrent, err := json.Marshal(createTorrent())
	verifyErrNil(err, t)
	return jsonTorrent
}

func createUserJson(t *testing.T) []byte {
	jsonUser, err := json.Marshal(createUser())
	verifyErrNil(err, t)
	return jsonUser
}

func createUserFromValues(userVals []string, t *testing.T) models.User {
	ID, err := strconv.ParseUint(userVals[0], 10, 64)
	verifyErrNil(err, t)
	Passkey := userVals[1]
	UpMultiplier, err := strconv.ParseFloat(userVals[2], 64)
	verifyErrNil(err, t)
	DownMultiplier, err := strconv.ParseFloat(userVals[3], 64)
	Slots, err := strconv.ParseInt(userVals[4], 10, 64)
	verifyErrNil(err, t)
	SlotsUsed, err := strconv.ParseInt(userVals[5], 10, 64)
	verifyErrNil(err, t)
	return models.User{ID, Passkey, UpMultiplier, DownMultiplier, Slots, SlotsUsed}
}
func createUser() models.User {
	testUser := models.User{214, "32426b162be0bce5428e7e36afaf734ae5afb355", 0.0, 0.0, 4, 2}
	return testUser
}

func createTorrent() models.Torrent {
	testSeeders := make([]models.Peer, 4)
	testSeeders[0] = models.Peer{"testPeerID0", 57005, 48879, "testIP", 6889, 1024, 3000, 4200, 6}
	testSeeders[1] = models.Peer{"testPeerID1", 10101, 48879, "testIP", 6889, 1024, 3000, 4200, 6}
	testSeeders[2] = models.Peer{"testPeerID2", 29890, 48879, "testIP", 6889, 1024, 3000, 4200, 6}
	testSeeders[3] = models.Peer{"testPeerID3", 65261, 48879, "testIP", 6889, 1024, 3000, 4200, 6}
	testLeechers := make([]models.Peer, 1)
	testLeechers[0] = models.Peer{"testPeerID", 11111, 48879, "testIP", 6889, 1024, 3000, 4200, 6}

	infohash := "58c290f4ea1efb3adcb8c1ed2643232117577bcd"

	seeders := make(map[string]models.Peer)
	for i := range testSeeders {
		seeders[testSeeders[i].ID] = testSeeders[i]
	}

	leechers := make(map[string]models.Peer)
	for i := range testLeechers {
		leechers[testLeechers[i].ID] = testLeechers[i]
	}

	testTorrent := models.Torrent{48879, infohash, true, seeders, leechers, 11, 0.0, 0.0, 0}
	return testTorrent
}

func TestRedisTransaction(t *testing.T) {
	for i := 0; i < 10; i++ {
		// No retries for serial transactions
		sampleTransaction(createTestTxObj(t), 0, t)
	}
}

func TestReadAfterWrite(t *testing.T) {
	testTx := createTestTxObj(t)

	verifyErrNil(testTx.initiateWrite(), t)

	// Test requires panic
	defer func() {
		if rawError := recover(); rawError == nil {
			t.Error("Read after write did not panic")
		}
	}()
	verifyErrNil(testTx.initiateRead(), t)
}

func TestDoubleClose(t *testing.T) {
	testTx := createTestTxObj(t)

	testTx.close()
	//require panic
	defer func() {
		if rawError := recover(); rawError == nil {
			t.Error("double close did not panic")
		}
	}()
	testTx.close()
}

func TestParallelTx0(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()

	for i := 0; i < 20; i++ {
		go sampleTransaction(createTestTxObj(t), MAX_RETRIES, t)
		time.Sleep(1 * time.Millisecond)
	}

}

func TestParallelTx1(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()
	sampleTransaction(createTestTxObj(t), MAX_RETRIES, t)
	for i := 0; i < 100; i++ {
		go sampleTransaction(createTestTxObj(t), MAX_RETRIES, t)
	}
}

func TestParallelTx2(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()
	for i := 0; i < 100; i++ {
		go sampleTransaction(createTestTxObj(t), MAX_RETRIES, t)
	}
	sampleTransaction(createTestTxObj(t), MAX_RETRIES, t)
}

// Just in case the above parallel tests didn't fail, force a failure here
func TestParallelInterrupted(t *testing.T) {
	t.Parallel()

	testTx := createTestTxObj(t)
	defer func() {
		if rawError := recover(); rawError != nil {
			t.Error("initiate read failed in parallelInterrupted")
		}
	}()
	verifyErrNil(testTx.initiateRead(), t)

	_, err := testTx.Do("WATCH", "testKeyA")
	verifyErrNil(err, t)

	testValueA, err := redis.String(testTx.Do("GET", "testKeyA"))
	if err != nil {
		if err == redis.ErrNil {
			t.Log("redis.ErrNil")
		} else {
			t.Error(err)
		}
	}

	_, err = testTx.Do("WATCH", "testKeyB")
	if err != nil {
		if err == redis.ErrNil {
			t.Log("redis.ErrNil")
		} else {
			t.Error(err)
		}
	}

	testValueB, err := redis.String(testTx.Do("GET", "testKeyB"))
	if err != nil {
		if err == redis.ErrNil {
			t.Log("redis.ErrNil")
		} else {
			t.Error(err)
		}
	}
	// Stand in for what real updates would do
	testValueB = testValueB + "+updates"
	testValueA = testValueA + "+updates"

	// Simulating another client interrupts transaction, causing exec to fail
	sampleTransaction(createTestTxObj(t), MAX_RETRIES, t)

	verifyErrNil(testTx.initiateWrite(), t)
	verifyErrNil(testTx.Send("SET", "testKeyA", testValueA), t)
	verifyErrNil(testTx.Send("SET", "testKeyB", testValueB), t)

	keys, err := (testTx.Do("EXEC"))
	// Expect error
	if keys != nil {
		t.Error("keys not nil, exec should have been interrupted")
	}
	verifyErrNil(err, t)

	testTx.close()
}
