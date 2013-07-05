// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package config

import (
	"strings"
	"testing"
)

var exampleConfig = `{

  "network": "tcp",
  "addr": ":34000",
  "storage": {
    "driver": "redis",
    "addr": "127.0.0.1:6379",
    "user": "root",
    "pass": "",
    "prefix": "test:",

    "max_idle_conn": 3,
    "idle_timeout": "240s",
    "conn_timeout": "5s"
  },

  "private": true,
  "freeleech": false,

  "announce": "30m",
  "min_announce": "15m",
  "read_timeout": "20s",
  "default_num_want": 50

}`

func TestNew(t *testing.T) {
	if _, err := New(strings.NewReader(exampleConfig)); err != nil {
		t.Error(err)
	}
}
