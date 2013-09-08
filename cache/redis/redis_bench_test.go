// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Benchmarks two different redis schemeas
package redis

import (
	"errors"
	"testing"

	"github.com/garyburd/redigo/redis"

	"github.com/pushrax/chihaya/models"
)

var (
	ErrTxDone     = errors.New("cache: Transaction has already been committed or rolled back")
	ErrTxConflict = errors.New("cache: Commit interrupted, update transaction and repeat")
)

// Maximum number of parallel retries; depends on system latency
const MAX_RETRIES = 9000

// Legacy JSON support for benching
func (tx *Tx) initiateWrite() error {
	if tx.done {
		return ErrTxDone
	}
	if tx.multi != true {
		tx.multi = true
		return tx.Send("MULTI")
	}
	return nil
}

func (tx *Tx) initiateRead() error {
	if tx.done {
		return ErrTxDone
	}
	if tx.multi == true {
		panic("Tried to read during MULTI")
	}
	return nil
}

func (tx *Tx) Commit() error {
	if tx.done {
		return ErrTxDone
	}
	if tx.multi == true {
		execResponse, err := tx.Do("EXEC")
		if execResponse == nil {
			tx.multi = false
			return ErrTxConflict
		}
		if err != nil {
			return err
		}
	}
	tx.close()
	return nil
}

func (tx *Tx) Rollback() error {
	if tx.done {
		return ErrTxDone
	}
	// Undoes watches and multi
	if _, err := tx.Do("DISCARD"); err != nil {
		return err
	}
	tx.multi = false
	tx.close()
	return nil
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

func BenchmarkRedisTypesSchemaRemoveSeeder(b *testing.B) {
	for bCount := 0; bCount < b.N; bCount++ {
		//TODO this needs to be updated
		b.Error("Unimplemented")

	}
}

func BenchmarkRedisTypesSchemaFindUser(b *testing.B) {

	// Ensure successful user find ( a failed lookup may have different performance )
	b.StopTimer()
	testUser := createTestUser()
	testTx := createTestTxObj(b)
	hashkey := testTx.conf.Prefix + UserPrefix + testUser.Passkey
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

		compareUser, exists := ExampleRedisTypesSchemaFindUser(testUser.Passkey, b)

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
