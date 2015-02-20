// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package chihaya

import (
	"flag"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/golang/glog"

	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/http"
	"github.com/chihaya/chihaya/stats"
	"github.com/chihaya/chihaya/tracker"

	// See the README for how to import custom drivers.
	_ "github.com/chihaya/chihaya/backend/noop"
)

var (
	maxProcs   int
	profile    string
	configPath string
)

func init() {
	flag.IntVar(&maxProcs, "maxprocs", runtime.NumCPU(), "maximum parallel threads")
	flag.StringVar(&profile, "profile", "", "if non-empty, path to write profiling data")
	flag.StringVar(&configPath, "config", "", "path to the configuration file")
}

// Boot starts Chihaya. By exporting this function, anyone can import their own
// custom drivers into their own package main and then call chihaya.Boot.
func Boot() {
	defer glog.Flush()

	flag.Parse()

	runtime.GOMAXPROCS(maxProcs)
	glog.V(1).Info("Set max threads to ", maxProcs)

	if profile != "" {
		f, err := os.Create(profile)
		if err != nil {
			glog.Fatalf("Failed to create profile file: %s\n", err)
		}
		defer f.Close()

		pprof.StartCPUProfile(f)
		glog.Info("Started profiling")

		defer func() {
			pprof.StopCPUProfile()
			glog.Info("Stopped profiling")
		}()
	}

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

	http.Serve(cfg, tkr)

	if err := tkr.Close(); err != nil {
		glog.Errorf("Failed to shut down tracker cleanly: %s", err.Error())
	}

	glog.Info("Gracefully shut down")
}
