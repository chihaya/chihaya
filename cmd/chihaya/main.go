// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"

	"github.com/chihaya/chihaya"
	"github.com/chihaya/chihaya/server"
	"github.com/chihaya/chihaya/tracker"

	// Servers
	_ "github.com/chihaya/chihaya/server/http"
	_ "github.com/chihaya/chihaya/server/prometheus"
	_ "github.com/chihaya/chihaya/server/store"
	_ "github.com/chihaya/chihaya/server/store/memory"

	// Middleware
	_ "github.com/chihaya/chihaya/middleware/deniability"
	_ "github.com/chihaya/chihaya/middleware/varinterval"
	_ "github.com/chihaya/chihaya/server/store/middleware/client"
	_ "github.com/chihaya/chihaya/server/store/middleware/infohash"
	_ "github.com/chihaya/chihaya/server/store/middleware/ip"
	_ "github.com/chihaya/chihaya/server/store/middleware/response"
	_ "github.com/chihaya/chihaya/server/store/middleware/swarm"
)

var (
	configPath string
	cpuprofile string
)

func init() {
	flag.StringVar(&configPath, "config", "", "path to the configuration file")
	flag.StringVar(&cpuprofile, "cpuprofile", "", "path to cpu profile output")
}

func main() {
	flag.Parse()

	if cpuprofile != "" {
		log.Println("profiling...")
		f, err := os.Create(cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	cfg, err := chihaya.OpenConfigFile(configPath)
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
