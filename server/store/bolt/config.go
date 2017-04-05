// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package bolt

import (
	"errors"

	"gopkg.in/yaml.v2"

	"github.com/chihaya/chihaya/server/store"
)

// ErrMissingFile is returned if no database file was specified.
var ErrMissingFile = errors.New("database file must not be empty")

// ErrMissingConfig is returned if a non-existent config is opened.
var ErrMissingConfig = errors.New("missing config")

type boltConfig struct {
	File string `yaml:"file"`
}

func newBoltConfig(storecfg *store.DriverConfig) (*boltConfig, error) {
	if storecfg == nil || storecfg.Config == nil {
		return nil, ErrMissingConfig
	}

	bytes, err := yaml.Marshal(storecfg.Config)
	if err != nil {
		return nil, err
	}

	var cfg boltConfig
	err = yaml.Unmarshal(bytes, &cfg)
	if err != nil {
		return nil, err
	}

	if cfg.File == "" {
		return nil, ErrMissingFile
	}

	return &cfg, nil
}
