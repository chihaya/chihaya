// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package prometheus implements a chihaya Server for serving metrics to
// Prometheus.
package prometheus

import (
	"errors"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/tylerb/graceful"
	"gopkg.in/yaml.v2"

	"github.com/chihaya/chihaya"
	"github.com/chihaya/chihaya/server"
	"github.com/chihaya/chihaya/tracker"
)

func init() {
	server.Register("prometheus", constructor)
}

func constructor(srvcfg *chihaya.ServerConfig, tkr *tracker.Tracker) (server.Server, error) {
	cfg, err := NewServerConfig(srvcfg)
	if err != nil {
		return nil, errors.New("prometheus: invalid config: " + err.Error())
	}

	return &Server{
		cfg: cfg,
	}, nil
}

// ServerConfig represents the configuration options for a
// PrometheusServer.
type ServerConfig struct {
	Addr            string        `yaml:"addr"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
}

// NewServerConfig marshals a chihaya.ServerConfig and unmarshals it
// into a more specific prometheus ServerConfig.
func NewServerConfig(srvcfg *chihaya.ServerConfig) (*ServerConfig, error) {
	bytes, err := yaml.Marshal(srvcfg.Config)
	if err != nil {
		return nil, err
	}

	var cfg ServerConfig
	err = yaml.Unmarshal(bytes, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Server implements a chihaya Server for serving metrics to Prometheus.
type Server struct {
	cfg     *ServerConfig
	grace   *graceful.Server
	stopped bool
}

var _ server.Server = &Server{}

func (s *Server) Start() {
	s.grace = &graceful.Server{
		Server: &http.Server{
			Addr:         s.cfg.Addr,
			Handler:      prometheus.Handler(),
			ReadTimeout:  s.cfg.ReadTimeout,
			WriteTimeout: s.cfg.WriteTimeout,
		},
		Timeout:          s.cfg.ShutdownTimeout,
		NoSignalHandling: true,
	}
}

func (s *Server) Stop() {
	s.grace.Stop(s.cfg.ShutdownTimeout)
	stopChan := s.grace.StopChan()

	// Block until the graceful server shuts down and closes this channel.
	for range stopChan {
	}
}
