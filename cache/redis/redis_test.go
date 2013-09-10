// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package redis

import (
	"crypto/rand"
	"io"
	"os"
	"strconv"
	"testing"

	"github.com/garyburd/redigo/redis"

	"github.com/pushrax/chihaya/config"
	"github.com/pushrax/chihaya/models"
)

var (
	testTorrentIDCounter uint64
	testUserIDCounter    uint64
	testPeerIDCounter    int
)

func createTestTorrentID() uint64 {
	testTorrentIDCounter++
	return testTorrentIDCounter
}

func createTestUserID() uint64 {
	testUserIDCounter++
	return testUserIDCounter
}

func createTestPeerID() string {
	testPeerIDCounter++
	return "-testPeerID-" + strconv.Itoa(testPeerIDCounter)
}

func createTestInfohash() string {
	uuid := make([]byte, 40)
	n, err := io.ReadFull(rand.Reader, uuid)
	if n != len(uuid) || err != nil {
		panic(err)
	}
	return string(uuid)
}

func createTestPasskey() string {
	uuid := make([]byte, 40)
	n, err := io.ReadFull(rand.Reader, uuid)
	if n != len(uuid) || err != nil {
		panic(err)
	}
	return string(uuid)
}

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
			MaxIdle:      conf.MaxIdleConns,
			IdleTimeout:  conf.IdleTimeout.Duration,
			Dial:         makeDialFunc(conf),
			TestOnBorrow: testOnBorrow,
		},
	}

	txObj := &Tx{
		conf:  testPool.conf,
		done:  false,
		multi: false,
		Conn:  testPool.pool.Get(),
	}
	verifyErrNil(err, t)

	// Test connection before returning
	_, err = txObj.Do("PING")
	verifyErrNil(err, t)
	return txObj
}

func createTestUser() models.User {
	testUser := models.User{createTestUserID(), createTestPasskey(), 1.01, 1.0, 4, 2, 7}
	return testUser
}

func createTestPeer(userID uint64, torrentID uint64) *models.Peer {

	return &models.Peer{createTestPeerID(), userID, torrentID, "127.0.0.1", 6889, 1024, 3000, 4200, 11}
}

func createTestPeers(torrentID uint64, num int) map[string]models.Peer {
	testPeers := make(map[string]models.Peer)
	for i := 0; i < num; i++ {
		tempPeer := createTestPeer(createTestUserID(), torrentID)
		testPeers[tempPeer.ID] = *tempPeer
	}
	return testPeers
}

func createTestTorrent() *models.Torrent {

	torrentInfohash := createTestInfohash()
	torrentID := createTestTorrentID()

	testSeeders := createTestPeers(torrentID, 4)
	testLeechers := createTestPeers(torrentID, 2)

	testTorrent := models.Torrent{torrentID, torrentInfohash, true, testSeeders, testLeechers, 11, 0.0, 0.0, 0}
	return &testTorrent
}

func TestAddGetPeers(t *testing.T) {

	testTx := createTestTxObj(t)
	testTorrent := createTestTorrent()

	setkey := testTx.conf.Prefix + SeederPrefix + strconv.FormatUint(testTorrent.ID, 36)
	testTx.Do("DEL", setkey)

	testTx.addPeers(testTorrent.Seeders, SeederPrefix)
	peerMap, err := testTx.getPeers(testTorrent.ID, SeederPrefix)
	if err != nil {
		t.Error(err)
	} else if len(peerMap) != len(testTorrent.Seeders) {
		t.Error("Num Peers not equal")
	}
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
