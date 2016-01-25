// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/server"
	"github.com/chihaya/chihaya/tracker"
)

var configPath string

func init() {
	flag.StringVar(&configPath, "config", "", "path to the configuration file")
}

func main() {
	flag.Parse()

	cfg, err := config.Open(configPath)
	if err != nil {
		log.Fatal("failed to load config: " + err.Error())
	}

	tkr, err := tracker.NewTracker(&cfg.Tracker)
	if err != nil {
		log.Fatal("failed to create tracker: " + err.Error())
	}

	pool, err := server.StartPool(cfg.Servers, tkr)
	if err != nil {
		log.Fatal("failed to create server pool: " + err.Error())
	}

	shutdown := make(chan os.Signal)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	<-shutdown
	pool.Stop()
}
