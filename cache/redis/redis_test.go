// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package redis implements the storage interface for a BitTorrent tracker.
// Benchmarks are at the top of the file, tests are at the bottom
package redis

import (
	"github.com/garyburd/redigo/redis"
	"github.com/pushrax/chihaya/cache"
	"github.com/pushrax/chihaya/config"
	"os"
	"testing"
	"time"
	"math/rand"
	"strconv"
)

var(
	//maximum number of parallel retries, will depends on system latency
	MAX_RETRIES = 9000
)

func CreateTestTxObj(t *testing.T) Tx {

	//assumes TESTCONFIGPATH has been defined
	testConfig, err := config.Open(os.Getenv("TESTCONFIGPATH"))
	if err != nil {
		t.Error(err)
	}

	testDialFunc := makeDialFunc(&testConfig.Cache)
	testConn, err := testDialFunc()
	if err != nil {
		t.Fail()
	}
	return Tx{&testConfig.Cache, false, false, testConn}

}

func SampleTransaction( testTx Tx, retries int, t *testing.T){

	defer func() {
		if rawError := recover(); rawError != nil {
			t.Errorf("initiate read failed")
			t.Fail()
		}
	} ()
	err := testTx.initiateRead()
	if err != nil {
		t.Fail()
	}
	_, err = testTx.Do("WATCH", "testKeyA")
	if err != nil {
		t.Errorf("error=%s", err)
	}
	_, err = redis.String(testTx.Do("GET", "testKeyA"))
	if err != nil {
		if err == redis.ErrNil {
			t.Log("redis.ErrNil")
		} else {
			t.Errorf("error=%s", err)
			t.Fail()
		}
	}
	_, err = testTx.Do("WATCH", "testKeyB")
	if err != nil {
		if err == redis.ErrNil {
			t.Log("redis.ErrNil")
		} else {
			t.Errorf("error=%s", err)
			t.Fail()
		}
	}
	_, err = redis.String(testTx.Do("GET", "testKeyB"))
	if err != nil {
		if err == redis.ErrNil {
			t.Log("redis.ErrNil")
		} else {
			t.Errorf("error=%s", err)
			t.Fail()
		}
	}

	err = testTx.initiateWrite()
	if err != nil {
		t.Errorf("error=%s", err)
		t.Fail()
	}

	//generate random data to set
	randGen := rand.New(rand.NewSource(time.Now().UnixNano()))
	err = testTx.Send("SET", "testKeyA", strconv.Itoa(randGen.Int()))
	if err != nil {
		t.Errorf("error=%s", err)
		t.Fail()
	}
	err = testTx.Send("SET", "testKeyB", strconv.Itoa(randGen.Int()))
	if err != nil {
		t.Errorf("error=%s", err)
		t.Fail()
	}

	err = testTx.Commit()
	//for parallel runs, there may be conflicts, retry until successful
	if err == cache.ErrTxConflict && retries > 0 {
		//t.Logf("Conflict, %d retries left",retries)
		SampleTransaction(testTx,retries-1,t)
		//clear TxConflict, if retries max out, errors are already recorded
		err = nil
	}else if err == cache.ErrTxConflict {
		t.Error("Conflict encountered, max retries reached")
		t.Errorf("error=%s", err)
	}
	if err != nil {
		t.Errorf("error=%s", err)
		t.Fail()
	}

}

func TestRedisTransaction(t *testing.T) {

	for i:=0; i < 10; i++ {
		//No retries for serial transactions
		SampleTransaction(CreateTestTxObj(t),0,t)
	}
}

func TestReadAfterWrite(t *testing.T) {

	testTx := CreateTestTxObj(t)

	err := testTx.initiateWrite()
	if err != nil {
		t.Errorf("error=%s", err)
		t.Fail()
	}
	// test requires panic
	defer func() {
		if rawError := recover(); rawError == nil {
			t.Errorf("Read after write did not panic")
			t.Fail()
		}
	} ()
	err = testTx.initiateRead()
	if err != nil {
		t.Errorf("error=%s", err)
		t.Fail()
	}
}

func TestDoubleClose(t *testing.T){

	testTx := CreateTestTxObj(t)

	testTx.close()
	//require panic
	defer func() {
		if rawError := recover(); rawError == nil {
			t.Errorf("double close did not panic")
			t.Fail()
		}
	} ()
	testTx.close()
}

func TestParallelTx0(t *testing.T) {

	t.Parallel()

	for i:=0; i< 20; i++ {
		go SampleTransaction(CreateTestTxObj(t),MAX_RETRIES,t)
		time.Sleep(1 * time.Millisecond)
	}

}

func TestParallelTx1(t *testing.T) {

	t.Parallel()
	SampleTransaction(CreateTestTxObj(t),MAX_RETRIES,t)
	for i:=0; i< 100; i++ {
		go SampleTransaction(CreateTestTxObj(t),MAX_RETRIES,t)
	}
}

func TestParallelTx2(t *testing.T) {

	t.Parallel()
	for i:=0; i< 100; i++ {
		go SampleTransaction(CreateTestTxObj(t),MAX_RETRIES,t)
	}
	SampleTransaction(CreateTestTxObj(t),MAX_RETRIES,t)
}

//just in case the above parallel tests didn't fail, force a failure here
func TestParallelInterrupted(t *testing.T) {

	t.Parallel()

	testTx := CreateTestTxObj(t)
	defer func() {
		if rawError := recover(); rawError != nil {
			t.Errorf("initiate read failed in parallelInterrupted")
			t.Fail()
		}
	} ()
	err := testTx.initiateRead()
	if err != nil {
		t.Fail()
	}

	_, err = testTx.Do("WATCH", "testKeyA")
	if err != nil {
		t.Errorf("error=%s", err)
	}

	testValueA, err := redis.String(testTx.Do("GET", "testKeyA"))
	if err != nil {
		if err == redis.ErrNil {
			t.Log("redis.ErrNil")
		} else {
			t.Errorf("error=%s", err)
			t.Fail()
		}
	}

	_, err = testTx.Do("WATCH", "testKeyB")
	if err != nil {
		if err == redis.ErrNil {
			t.Log("redis.ErrNil")
		} else {
			t.Errorf("error=%s", err)
			t.Fail()
		}
	}

	testValueB, err := redis.String(testTx.Do("GET", "testKeyB"))
	if err != nil {
		if err == redis.ErrNil {
			t.Log("redis.ErrNil")
		} else {
			t.Errorf("error=%s", err)
			t.Fail()
		}
	}
	//stand in for what real updates would do
	testValueB = testValueB + "+updates"
	testValueA = testValueA + "+updates"

	// simulating another client interrupts transaction, causing exec to fail
	SampleTransaction(CreateTestTxObj(t),MAX_RETRIES,t)

	err = testTx.initiateWrite()
	if err != nil {
		t.Errorf("error=%s", err)
		t.Fail()
	}
	err = testTx.Send("SET", "testKeyA", testValueA)
	if err != nil {
		t.Errorf("error=%s", err)
		t.Fail()
	}

	err = testTx.Send("SET", "testKeyB", testValueB)
	if err != nil {
		t.Errorf("error=%s", err)
		t.Fail()
	}

	keys, err := (testTx.Do("EXEC"))
	//expect error
	if keys != nil {
		t.Errorf("keys not nil, exec should have been interrupted")
	}

	testTx.close()
	if err != nil {
		t.Errorf("error=%s", err)
		t.Fail()
	}
}
