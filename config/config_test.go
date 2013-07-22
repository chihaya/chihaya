// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package config

import (
	"bufio"
	"os"
	"strings"
	"testing"
)

var exampleJson = `{

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

func TestNewConfig(t *testing.T) {
	if _, err := newConfig(strings.NewReader(exampleJson)); err != nil {
		t.Error(err)
	}
}

func writeAndOpenJsonTest(t *testing.T, fn string) {
	expandFn := os.ExpandEnv(fn)
	// Write JSON to relative path, clean up
	tfile, ferr := os.Create(expandFn)
	// Remove failure not counted as error
	defer os.Remove(expandFn)
	if ferr != nil {
		t.Fatal("Failed to create %s. Error: %v", expandFn, ferr)
	}

	tWriter := bufio.NewWriter(tfile)
	cw, err := tWriter.WriteString(exampleJson)
	if err != nil {
		t.Fatal("Failed to write json to config file. %v", err)
	}
	if cw < len(exampleJson) {
		t.Error("Incorrect length of config file written %v vs. %v", cw, len(exampleJson))
	}
	fErr := tWriter.Flush()
	if fErr != nil {
		t.Error("Flush error: %v", fErr)
	}
	_, oErr := Open(fn)
	if oErr != nil {
		t.Error("Open error: %v", oErr)
	}
}

// These implcitly require the test program have
// read/write/delete file system permissions
func TestOpenCurDir(t *testing.T) {
	if !testing.Short() {
		writeAndOpenJsonTest(t, "testConfig.json")
	} else {
		t.Log("Write/Read file test skipped")
	}
}
func TestOpenAbsEnvPath(t *testing.T) {
	if !testing.Short() {
		writeAndOpenJsonTest(t, os.TempDir()+"testConfig.json")
	} else {
		t.Log("Write/Read file test skipped")
	}
}
