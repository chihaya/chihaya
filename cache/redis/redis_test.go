// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

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

func createTestTxObj(t TestReporter) *Tx {
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

func createUserFromValues(userVals []string, t TestReporter) *models.User {
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
	return &models.User{ID, Passkey, UpMultiplier, DownMultiplier, Slots, SlotsUsed}
}

func createUser() models.User {
	testUser := models.User{214, "32426b162be0bce5428e7e36afaf734ae5afb355", 0.0, 0.0, 4, 2}
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

func createTorrent() models.Torrent {

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

func createTorrentJson(t TestReporter) []byte {
	jsonTorrent, err := json.Marshal(createTorrent())
	verifyErrNil(err, t)
	return jsonTorrent
}

func createUserJson(t TestReporter) []byte {
	jsonUser, err := json.Marshal(createUser())
	verifyErrNil(err, t)
	return jsonUser
}

func ExampleJsonTransaction(testTx *Tx, retries int, t TestReporter) {
	defer func() {
		if err := recover(); err != nil {
			t.Error(err)
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
	verifyErrNil(err, t)

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
		ExampleJsonTransaction(testTx, retries-1, t)
		// Clear TxConflict, if retries max out, errors are already recorded
		err = nil
	} else if err == cache.ErrTxConflict {
		t.Error("Conflict encountered, max retries reached")
		t.Error(err)
	}
	verifyErrNil(err, t)
}

func ExampleJsonSchemaRemoveSeeder(torrent *models.Torrent, peer *models.Peer, t TestReporter) {
	testTx := createTestTxObj(t)

	verifyErrNil(testTx.initiateRead(), t)

	key := testTx.conf.Prefix + "torrent:" + torrent.Infohash
	_, err := testTx.Do("WATCH", key)
	reply, err := redis.String(testTx.Do("GET", key))
	if err != nil {
		if err == redis.ErrNil {
			t.Error("testTorrent does not exist")
		} else {
			t.Error(err)
		}
	}

	verifyErrNil(json.NewDecoder(strings.NewReader(reply)).Decode(torrent), t)

	delete(torrent.Seeders, "testPeerID2")

	jsonTorrent, err := json.Marshal(torrent)
	verifyErrNil(err, t)
	verifyErrNil(testTx.initiateWrite(), t)
	verifyErrNil(testTx.Send("SET", key, jsonTorrent), t)
	verifyErrNil(testTx.Commit(), t)

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

func ExampleJsonSchemaFindUser(passkey string, t TestReporter) (*models.User, bool) {
	testTx := createTestTxObj(t)

	verifyErrNil(testTx.initiateRead(), t)

	key := testTx.conf.Prefix + "user:" + passkey
	_, err := testTx.Do("WATCH", key)
	verifyErrNil(err, t)
	reply, err := redis.String(testTx.Do("GET", key))
	if err != nil {
		if err == redis.ErrNil {
			return nil, false
		} else {
			t.Error(err)
		}
	}

	user := &models.User{}
	verifyErrNil(json.NewDecoder(strings.NewReader(reply)).Decode(user), t)
	return user, true
}

func ExampleRedisTypesSchemaFindUser(passkey string, t TestReporter) (*models.User, bool) {
	testTx := createTestTxObj(t)
	hashkey := testTx.conf.Prefix + "user_hash:" + sample_passkey
	userVals, err := redis.Strings(testTx.Do("HVALS", hashkey))
	if userVals == nil {
		return nil, false
	}
	verifyErrNil(err, t)
	compareUser := createUserFromValues(userVals, t)
	return compareUser, true
}

func BenchmarkRedisJsonSchemaRemoveSeeder(b *testing.B) {
	for bCount := 0; bCount < b.N; bCount++ {
		b.StopTimer()
		testTx := createTestTxObj(b)
		testTorrent := createTorrent()
		testSeeders := createSeeders()
		key := testTx.conf.Prefix + "torrent:" + testTorrent.Infohash
		// Benchmark setup not a transaction, not thread-safe
		_, err := testTx.Do("SET", key, createTorrentJson(b))
		verifyErrNil(err, b)
		b.StartTimer()

		ExampleJsonSchemaRemoveSeeder(&testTorrent, &testSeeders[2], b)
	}
}

func BenchmarkRedisTypesSchemaRemoveSeeder(b *testing.B) {
	for bCount := 0; bCount < b.N; bCount++ {

		// Ensure that remove completes successfully,
		// even if it doesn't impact the performance
		b.StopTimer()
		testTx := createTestTxObj(b)
		testTorrent := createTorrent()
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

func BenchmarkRedisJsonSchemaFindUser(b *testing.B) {
	// Ensure successful user find ( a failed lookup may have different performance )
	b.StopTimer()
	testTx := createTestTxObj(b)
	testUser := createUser()
	userJson := string(createUserJson(b))
	verifyErrNil(testTx.initiateWrite(), b)
	key := testTx.conf.Prefix + "user:" + sample_passkey
	verifyErrNil(testTx.Send("SET", key, userJson), b)
	verifyErrNil(testTx.Commit(), b)
	b.StartTimer()
	for bCount := 0; bCount < b.N; bCount++ {
		compareUser, exists := ExampleJsonSchemaFindUser(sample_passkey, b)
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

func BenchmarkRedisTypesSchemaFindUser(b *testing.B) {

	// Ensure successful user find ( a failed lookup may have different performance )
	b.StopTimer()
	testUser := createUser()
	testTx := createTestTxObj(b)
	hashkey := testTx.conf.Prefix + "user_hash:" + sample_passkey
	reply, err := testTx.Do("HMSET", hashkey,
		"id", testUser.ID,
		"passkey", testUser.Passkey,
		"up_multiplier", testUser.UpMultiplier,
		"down_multiplier", testUser.DownMultiplier,
		"slots", testUser.Slots,
		"slots_used", testUser.SlotsUsed)

	if reply == nil {
		b.Error("no hash fields added!")
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

func TestRedisTransaction(t *testing.T) {
	for i := 0; i < 10; i++ {
		// No retries for serial transactions
		ExampleJsonTransaction(createTestTxObj(t), 0, t)
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

func TestParallelTx0(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()

	for i := 0; i < 20; i++ {
		go ExampleJsonTransaction(createTestTxObj(t), MAX_RETRIES, t)
		time.Sleep(1 * time.Millisecond)
	}

}

func TestParallelTx1(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()
	ExampleJsonTransaction(createTestTxObj(t), MAX_RETRIES, t)
	for i := 0; i < 100; i++ {
		go ExampleJsonTransaction(createTestTxObj(t), MAX_RETRIES, t)
	}
}

func TestParallelTx2(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()
	for i := 0; i < 100; i++ {
		go ExampleJsonTransaction(createTestTxObj(t), MAX_RETRIES, t)
	}
	ExampleJsonTransaction(createTestTxObj(t), MAX_RETRIES, t)
}

// Just in case the above parallel tests didn't fail, force a failure here
func TestParallelInterrupted(t *testing.T) {
	t.Parallel()

	testTx := createTestTxObj(t)
	defer func() {
		if err := recover(); err != nil {
			t.Errorf("initiateRead() failed in parallel %s", err)
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
	ExampleJsonTransaction(createTestTxObj(t), MAX_RETRIES, t)

	verifyErrNil(testTx.initiateWrite(), t)
	verifyErrNil(testTx.Send("SET", "testKeyA", testValueA), t)
	verifyErrNil(testTx.Send("SET", "testKeyB", testValueB), t)

	keys, err := (testTx.Do("EXEC"))
	// Expect error
	if keys != nil {
		t.Error("Keys not nil; exec should have been interrupted")
	}
	verifyErrNil(err, t)

	testTx.close()
}
