// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package http

import (
	"time"

	"gopkg.in/yaml.v2"
)

type httpConfig struct {
	Addr             string        `yaml:"addr"`
	RequestTimeout   time.Duration `yaml:"requestTimeout"`
	ReadTimeout      time.Duration `yaml:"readTimeout"`
	WriteTimeout     time.Duration `yaml:"writeTimeout"`
	AllowIPSpoofing  bool          `yaml:"allowIPSpoofing"`
	DualStackedPeers bool          `yaml:"dualStackedPeers"`
	RealIPHeader     string        `yaml:"realIPHeader"`
}

func newHTTPConfig(srvcfg interface{}) (*httpConfig, error) {
	bytes, err := yaml.Marshal(srvcfg)
	if err != nil {
		return nil, err
	}

	var cfg httpConfig
	err = yaml.Unmarshal(bytes, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
