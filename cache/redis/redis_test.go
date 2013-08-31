// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package redis

import (
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/garyburd/redigo/redis"

	"github.com/pushrax/chihaya/cache"
	"github.com/pushrax/chihaya/config"
)

// Maximum number of parallel retries; depends on system latency
const MAX_RETRIES = 9000

func CreateTestTxObj(t *testing.T) Tx {
	testConfig, err := config.Open(os.Getenv("TESTCONFIGPATH"))
	if err != nil {
		t.Error(err)
	}

	testDialFunc := makeDialFunc(&testConfig.Cache)
	testConn, err := testDialFunc()
	if err != nil {
		t.Error(err)
	}
	return Tx{&testConfig.Cache, false, false, testConn}
}

func SampleTransaction(testTx Tx, retries int, t *testing.T) {
	defer func() {
		if err := recover(); err != nil {
			t.Errorf("initiateRead() failed: %s", err)
		}
	}()
	err := testTx.initiateRead()
	if err != nil {
		t.Error(err)
	}
	_, err = testTx.Do("WATCH", "testKeyA")
	if err != nil {
		t.Error(err)
	}
	_, err = redis.String(testTx.Do("GET", "testKeyA"))
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
	_, err = redis.String(testTx.Do("GET", "testKeyB"))
	if err != nil {
		if err == redis.ErrNil {
			t.Log("redis.ErrNil")
		} else {
			t.Error(err)
		}
	}

	err = testTx.initiateWrite()
	if err != nil {
		t.Error(err)
	}

	// Generate random data to set
	randGen := rand.New(rand.NewSource(time.Now().UnixNano()))
	err = testTx.Send("SET", "testKeyA", strconv.Itoa(randGen.Int()))
	if err != nil {
		t.Error(err)
	}
	err = testTx.Send("SET", "testKeyB", strconv.Itoa(randGen.Int()))
	if err != nil {
		t.Error(err)
	}

	err = testTx.Commit()
	switch {
	// For parallel runs, there may be conflicts, retry until successful
	case err == cache.ErrTxConflict && retries > 0:
		// t.Logf("Conflict, %d retries left",retries)
		SampleTransaction(testTx, retries-1, t)
		// Clear TxConflict, if retries max out, errors are already recorded
		err = nil
	case err == cache.ErrTxConflict:
		t.Errorf("Conflict encountered, max retries reached: %s", err)
	case err != nil:
		t.Error(err)
	default:
	}
}

func TestRedisTransaction(t *testing.T) {
	for i := 0; i < 10; i++ {
		// No retries for serial transactions
		SampleTransaction(CreateTestTxObj(t), 0, t)
	}
}

func TestReadAfterWrite(t *testing.T) {
	defer func() {
		if err := recover(); err == nil {
			t.Error("Read after write did not panic")
		}
	}()

	testTx := CreateTestTxObj(t)
	err := testTx.initiateWrite()
	if err != nil {
		t.Error(err)
	}

	err = testTx.initiateRead()
	if err != nil {
		t.Error(err)
	}
}

func TestCloseClosedTransaction(t *testing.T) {
	defer func() {
		if err := recover(); err == nil {
			t.Error("Closing a closed transaction did not panic")
		}
	}()

	testTx := CreateTestTxObj(t)
	testTx.close()
	testTx.close()
}

func TestParallelTx0(t *testing.T) {
	t.Parallel()
	for i := 0; i < 20; i++ {
		go SampleTransaction(CreateTestTxObj(t), MAX_RETRIES, t)
		time.Sleep(1 * time.Millisecond)
	}
}

func TestParallelTx1(t *testing.T) {
	t.Parallel()
	SampleTransaction(CreateTestTxObj(t), MAX_RETRIES, t)
	for i := 0; i < 100; i++ {
		go SampleTransaction(CreateTestTxObj(t), MAX_RETRIES, t)
	}
}

func TestParallelTx2(t *testing.T) {
	t.Parallel()
	for i := 0; i < 100; i++ {
		go SampleTransaction(CreateTestTxObj(t), MAX_RETRIES, t)
	}
	SampleTransaction(CreateTestTxObj(t), MAX_RETRIES, t)
}

// Just in case the above parallel tests didn't fail, force a failure here
func TestParallelInterrupted(t *testing.T) {
	t.Parallel()

	defer func() {
		if err := recover(); err != nil {
			t.Errorf("initiateRead() failed in parallel: %s", err)
		}
	}()

	testTx := CreateTestTxObj(t)
	err := testTx.initiateRead()
	if err != nil {
		t.Error(err)
	}

	_, err = testTx.Do("WATCH", "testKeyA")
	if err != nil {
		t.Error(err)
	}

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
	SampleTransaction(CreateTestTxObj(t), MAX_RETRIES, t)

	err = testTx.initiateWrite()
	if err != nil {
		t.Error(err)
	}
	err = testTx.Send("SET", "testKeyA", testValueA)
	if err != nil {
		t.Error(err)
	}

	err = testTx.Send("SET", "testKeyB", testValueB)
	if err != nil {
		t.Error(err)
	}

	keys, err := (testTx.Do("EXEC"))
	if keys != nil {
		t.Error("Keys not nil; exec should have been interrupted")
	}

	testTx.close()
	if err != nil {
		t.Error(err)
	}
}
