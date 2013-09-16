// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package redis

import (
	"crypto/rand"
	"fmt"
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

func panicErrNil(err error) {
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
}

func createTestTxObj() *Tx {
	testConfig, err := config.Open(os.Getenv("TESTCONFIGPATH"))
	conf := &testConfig.Cache
	panicErrNil(err)

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
		conf: testPool.conf,
		done: false,
		Conn: testPool.pool.Get(),
	}
	panicErrNil(err)

	// Test connection before returning
	_, err = txObj.Do("PING")
	panicErrNil(err)
	return txObj
}

func createTestUser() *models.User {
	return &models.User{ID: createTestUserID(), Passkey: createTestPasskey(),
		UpMultiplier: 1.01, DownMultiplier: 1.0, Slots: 4, SlotsUsed: 2, Snatches: 7}
}

func createTestPeer(userID uint64, torrentID uint64) *models.Peer {

	return &models.Peer{ID: createTestPeerID(), UserID: userID, TorrentID: torrentID,
		IP: "127.0.0.1", Port: 6889, Uploaded: 1024, Downloaded: 3000, Left: 4200, LastAnnounce: 11}
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

	testTorrent := models.Torrent{ID: torrentID, Infohash: torrentInfohash, Active: true,
		Seeders: testSeeders, Leechers: testLeechers, Snatches: 11, UpMultiplier: 1.0, DownMultiplier: 1.0, LastAction: 0}
	return &testTorrent
}

func TestPeersAlone(t *testing.T) {

	testTx := createTestTxObj()
	testTorrentID := createTestTorrentID()
	testPeers := createTestPeers(testTorrentID, 3)

	panicErrNil(testTx.addPeers(testPeers, "test:"))
	peerMap, err := testTx.getPeers(testTorrentID, "test:")
	panicErrNil(err)
	if len(peerMap) != len(testPeers) {
		t.Error("Num Peers not equal ", len(peerMap), len(testPeers))
	}
	panicErrNil(testTx.removePeers(testTorrentID, testPeers, "test:"))
}
