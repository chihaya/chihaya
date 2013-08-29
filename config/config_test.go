// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package config

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"
)

func getConfigPath() string {
	if os.Getenv("TRAVISCONFIGPATH") != "" {
		return os.Getenv("TRAVISCONFIGPATH")
	}
	return os.ExpandEnv("$GOPATH/src/github.com/pushrax/chihaya/config/example.json")
}

func TestOpenConfig(t *testing.T) {
	if _, err := Open(getConfigPath()); err != nil {
		t.Error(err)
	}
}

func TestNewConfig(t *testing.T) {
	contents, err := ioutil.ReadFile(getConfigPath())
	if err != nil {
		t.Error(err)
	}
	buff := bytes.NewBuffer(contents)
	if _, err := newConfig(buff); err != nil {
		t.Error(err)
	}
}
