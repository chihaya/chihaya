// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package chihaya implements the ability to boot the Chihaya BitTorrent
// tracker with your own imports that can dynamically register additional
// functionality.
package chihaya

import (
	"flag"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"

	"github.com/golang/glog"

	"github.com/chihaya/chihaya/api"
	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/http"
	"github.com/chihaya/chihaya/stats"
	"github.com/chihaya/chihaya/tracker"
	"github.com/chihaya/chihaya/udp"

	// See the README for how to import custom drivers.
	_ "github.com/chihaya/chihaya/backend/noop"
)

var (
	maxProcs   int
	configPath string
)

func init() {
	flag.IntVar(&maxProcs, "maxprocs", runtime.NumCPU(), "maximum parallel threads")
	flag.StringVar(&configPath, "config", "", "path to the configuration file")
}

// Boot starts Chihaya. By exporting this function, anyone can import their own
// custom drivers into their own package main and then call chihaya.Boot.
func Boot() {
	defer glog.Flush()

	flag.Parse()

	runtime.GOMAXPROCS(maxProcs)
	glog.V(1).Info("Set max threads to ", maxProcs)

	debugBoot()
	defer debugShutdown()

	cfg, err := config.Open(configPath)
	if err != nil {
		glog.Fatalf("Failed to parse configuration file: %s\n", err)
	}

	if cfg == &config.DefaultConfig {
		glog.V(1).Info("Using default config")
	} else {
		glog.V(1).Infof("Loaded config file: %s", configPath)
	}

	stats.DefaultStats = stats.New(cfg.StatsConfig)

	tkr, err := tracker.New(cfg)
	if err != nil {
		glog.Fatal("New: ", err)
	}

	var servers []interface {
		Serve()
		Stop()
	}

	if cfg.APIConfig.ListenAddr != "" {
		srv := api.NewServer(cfg, tkr)
		servers = append(servers, srv)
	}

	if cfg.HTTPConfig.ListenAddr != "" {
		srv := http.NewServer(cfg, tkr)
		servers = append(servers, srv)
	}

	if cfg.UDPConfig.ListenAddr != "" {
		srv := udp.NewServer(cfg, tkr)
		servers = append(servers, srv)
	}

	var wg sync.WaitGroup
	for _, srv := range servers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			srv.Serve()
		}()
	}

	shutdown := make(chan os.Signal)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		wg.Wait()
		signal.Stop(shutdown)
		close(shutdown)
	}()

	<-shutdown
	glog.Info("Shutting down...")

	for _, srv := range servers {
		srv.Stop()
	}

	<-shutdown

	if err := tkr.Close(); err != nil {
		glog.Errorf("Failed to shut down tracker cleanly: %s", err.Error())
	}
}
