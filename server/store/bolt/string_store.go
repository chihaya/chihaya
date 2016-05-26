// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package bolt

import (
	"errors"
	"time"

	"github.com/boltdb/bolt"

	"github.com/chihaya/chihaya/server/store"
)

func init() {
	store.RegisterStringStoreDriver("bolt", &stringStoreDriver{})
}

// ErrMissingBucket is returned if a required bucket does not exist.
var ErrMissingBucket = errors.New("missing bucket")

const val = ""
const stringsBucketKey = "strings"

type stringStoreDriver struct{}

func (d *stringStoreDriver) New(storecfg *store.DriverConfig) (store.StringStore, error) {
	cfg, err := newBoltConfig(storecfg)
	if err != nil {
		return nil, err
	}

	db, err := bolt.Open(cfg.File, 0600, &bolt.Options{Timeout: 5 * time.Second})
	if err != nil {
		return nil, err
	}

	err = db.Update(func(tx *bolt.Tx) error {
		_, err = tx.CreateBucketIfNotExists([]byte(stringsBucketKey))
		return err
	})
	if err != nil {
		return nil, err
	}

	return &stringStore{
		db: db,
	}, nil
}

type stringStore struct {
	db *bolt.DB
}

func (s *stringStore) PutString(str string) error {
	err := s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(stringsBucketKey))
		if b == nil {
			return ErrMissingBucket
		}

		return b.Put([]byte(str), []byte(val))
	})

	return err
}

func (s *stringStore) HasString(str string) (contained bool, err error) {
	err = s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(stringsBucketKey))
		if b == nil {
			return ErrMissingBucket
		}

		contained = b.Get([]byte(str)) != nil

		return nil
	})

	return
}

func (s *stringStore) RemoveString(str string) error {
	err := s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(stringsBucketKey))
		if b == nil {
			return ErrMissingBucket
		}

		if b.Get([]byte(str)) == nil {
			return store.ErrResourceDoesNotExist
		}

		return b.Delete([]byte(str))
	})

	return err
}

func (s *stringStore) Stop() <-chan error {
	toReturn := make(chan error)
	go func() {
		err := s.db.Close()
		if err != nil {
			toReturn <- err
		}
		close(toReturn)
	}()
	return toReturn
}
