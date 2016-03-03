// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package http

import (
	"time"

	"gopkg.in/yaml.v2"

	"github.com/chihaya/chihaya"
)

type httpConfig struct {
	Addr             string        `yaml:"addr"`
	RequestTimeout   time.Duration `yaml:"request_timeout"`
	ReadTimeout      time.Duration `yaml:"read_timeout"`
	WriteTimeout     time.Duration `yaml:"write_timeout"`
	AllowIPSpoofing  bool          `yaml:"allow_ip_spoofing"`
	DualStackedPeers bool          `yaml:"dual_stacked_peers"`
	RealIPHeader     string        `yaml:"real_ip_header"`
}

func newHTTPConfig(srvcfg *chihaya.ServerConfig) (*httpConfig, error) {
	bytes, err := yaml.Marshal(srvcfg.Config)
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
