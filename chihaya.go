// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package chihaya

import (
	"flag"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"

	"github.com/golang/glog"

	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/stats"
	"github.com/chihaya/chihaya/tracker"
	"github.com/chihaya/chihaya/transport/http"
	"github.com/chihaya/chihaya/transport/udp"

	// Import our default storage drivers.
	_ "github.com/chihaya/chihaya/deltastore/nop"
	_ "github.com/chihaya/chihaya/store/memory"
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

	var wg sync.WaitGroup
	var servers []tracker.Server

	if cfg.HTTPListenAddr != "" {
		wg.Add(1)
		srv := http.NewServer(cfg, tkr)
		servers = append(servers, srv)

		go func() {
			defer wg.Done()
			srv.Serve(cfg.HTTPListenAddr)
		}()
	}

	if cfg.UDPListenAddr != "" {
		wg.Add(1)
		srv := udp.NewServer(cfg, tkr)
		servers = append(servers, srv)

		go func() {
			defer wg.Done()
			srv.Serve(cfg.UDPListenAddr)
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
