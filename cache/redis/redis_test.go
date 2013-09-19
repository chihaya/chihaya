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
	testTorrentIDChannel chan uint64
	testUserIDChannel    chan uint64
	testPeerIDChannel    chan int
)

func init() {
	testTorrentIDChannel = make(chan uint64, 100)
	testUserIDChannel = make(chan uint64, 100)
	testPeerIDChannel = make(chan int, 100)
	// Sync access to ID counter with buffered global channels
	go func() {
		for i := 0; ; i++ {
			testTorrentIDChannel <- uint64(i)
		}
	}()
	go func() {
		for i := 0; ; i++ {
			testUserIDChannel <- uint64(i)
		}
	}()
	go func() {
		for i := 0; ; i++ {
			testPeerIDChannel <- i
		}
	}()
}

func createTestTorrentID() uint64 {
	return <-testTorrentIDChannel
}

func createTestUserID() uint64 {
	return <-testUserIDChannel
}

func createTestPeerID() string {
	return "-testPeerID-" + strconv.Itoa(<-testPeerIDChannel)
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

func panicErrNil(err error) {
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
}

func createTestRedisTx() *Tx {
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
		testPeers[models.PeerMapKey(tempPeer)] = *tempPeer
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

func comparePeers(lhPeers map[string]models.Peer, rhPeers map[string]models.Peer) bool {
	if len(lhPeers) != len(rhPeers) {
		return false
	}
	for rhKey, rhValue := range rhPeers {
		lhValue, lhExists := lhPeers[rhKey]
		if !lhExists || lhValue != rhValue {
			return false
		}
	}
	for lhKey, lhValue := range lhPeers {
		rhValue, rhExists := rhPeers[lhKey]
		if !rhExists || rhValue != lhValue {
			return false
		}
	}
	return true
}

func torrentsEqual(lhTorrent *models.Torrent, rhTorrent *models.Torrent) bool {
	fieldsEqual := lhTorrent.Infohash == rhTorrent.Infohash &&
		lhTorrent.ID == rhTorrent.ID &&
		lhTorrent.Active == rhTorrent.Active &&
		lhTorrent.Snatches == rhTorrent.Snatches &&
		lhTorrent.UpMultiplier == rhTorrent.UpMultiplier &&
		lhTorrent.DownMultiplier == rhTorrent.DownMultiplier &&
		lhTorrent.LastAction == rhTorrent.LastAction

	if !fieldsEqual {
		return false
	}

	return comparePeers(lhTorrent.Seeders, rhTorrent.Seeders) && comparePeers(lhTorrent.Leechers, rhTorrent.Leechers)
}

func TestValidPeers(t *testing.T) {
	testTx := createTestRedisTx()
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

func TestInvalidPeers(t *testing.T) {
	testTx := createTestRedisTx()
	testTorrentID := createTestTorrentID()
	testPeers := createTestPeers(testTorrentID, 3)
	tempPeer := createTestPeer(createTestUserID(), testTorrentID)
	testPeers[models.PeerMapKey(tempPeer)] = *tempPeer

	panicErrNil(testTx.addPeers(testPeers, "test:"))
	// Imitate a peer being removed during get
	hashKey := testTx.conf.Prefix + getPeerHashKey(tempPeer)
	_, err := testTx.Do("DEL", hashKey)
	panicErrNil(err)

	peerMap, err := testTx.getPeers(testTorrentID, "test:")
	panicErrNil(err)
	// Expect 1 less peer due to delete
	if len(peerMap) != len(testPeers)-1 {
		t.Error("Num Peers not equal ", len(peerMap), len(testPeers))
	}
	panicErrNil(testTx.removePeers(testTorrentID, testPeers, "test:"))
}
