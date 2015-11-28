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
	"log"

	"github.com/chihaya/chihaya/api"
	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/http"
	"github.com/chihaya/chihaya/stats"
	"github.com/chihaya/chihaya/tracker"
	"github.com/chihaya/chihaya/udp"

	"github.com/mrd0ll4r/logger"
)

var (
	maxProcs        int
	configPath      string
	logLevel        int
	logFileLocation bool
)

func init() {
	flag.IntVar(&maxProcs, "maxprocs", runtime.NumCPU(), "maximum parallel threads")
	flag.StringVar(&configPath, "config", "", "path to the configuration file")
	flag.IntVar(&logLevel, "level", int(logger.LevelInfo), "level of logging, where 0=Everything, 1=Trace, 2=Debug, 3=Info, 4=Warnings, 5=Fatal")
	flag.BoolVar(&logFileLocation, "logLocation", false, "whether to log file locations")
}

type server interface {
	Serve()
	Stop()
}

// Boot starts Chihaya. By exporting this function, anyone can import their own
// custom drivers into their own package main and then call chihaya.Boot.
func Boot() {
	flag.Parse()

	l := logger.NewStdlibLogger()
	if logFileLocation {
		l.SetFlags(log.Lshortfile | log.Lmicroseconds | log.Ldate)
	} else {
		l.SetFlags(log.Lmicroseconds | log.Ldate)
	}

	level := logger.LogLevel(logLevel)

	if level < logger.Everything || level >= logger.Off {
		l.Fatalln("Invalid log level")
	}
	l.SetLevel(level)

	// we have to do this to use the short functions like logger.Infoln
	l.SetCalldepthForDefault()

	logger.SetDefaultLogger(l)

	runtime.GOMAXPROCS(maxProcs)
	logger.Debugf("Set max threads to %d", maxProcs)

	debugBoot()
	defer debugShutdown()

	cfg, err := config.Open(configPath)
	if err != nil {
		logger.Fatalf("Failed to parse configuration file: %s", err)
	}

	if cfg == &config.DefaultConfig {
		logger.Infoln("Using default config")
	} else {
		logger.Infof("Loaded config file %s", configPath)
	}

	stats.DefaultStats = stats.New(cfg.StatsConfig)

	tkr, err := tracker.New(cfg)
	if err != nil {
		logger.Fatalln("Unable to create new tracker:", err)
	}

	var servers []server

	if cfg.APIConfig.ListenAddr != "" {
		servers = append(servers, api.NewServer(cfg, tkr))
	}

	if cfg.HTTPConfig.ListenAddr != "" {
		servers = append(servers, http.NewServer(cfg, tkr))
	}

	if cfg.UDPConfig.ListenAddr != "" {
		servers = append(servers, udp.NewServer(cfg, tkr))
	}

	var wg sync.WaitGroup
	for _, srv := range servers {
		wg.Add(1)

		// If you don't explicitly pass the server, every goroutine captures the
		// last server in the list.
		go func(srv server) {
			defer wg.Done()
			srv.Serve()
		}(srv)
	}

	shutdown := make(chan os.Signal)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		wg.Wait()
		signal.Stop(shutdown)
		close(shutdown)
	}()

	<-shutdown
	logger.Infoln("Shutting down...")

	for _, srv := range servers {
		srv.Stop()
	}

	<-shutdown

	if err := tkr.Close(); err != nil {
		logger.Warnf("Failed to shut down tracker cleanly: %s", err.Error())
	}
}
