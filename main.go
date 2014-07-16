// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package main

import (
	"flag"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/golang/glog"

	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/http"

	// All drivers are imported here.
	_ "github.com/chihaya/chihaya/drivers/backend/noop"
	_ "github.com/chihaya/chihaya/drivers/tracker/memory"
)

var (
	maxprocs   int
	profile    string
	configPath string
)

func init() {
	flag.IntVar(&maxprocs, "maxprocs", runtime.NumCPU(), "maximum parallel threads")
	flag.StringVar(&profile, "profile", "", "if non-empty, path to write profiling data")
	flag.StringVar(&configPath, "config", "", "path to the configuration file")
}

func Boot() {
	defer glog.Flush()

	flag.Parse()

	runtime.GOMAXPROCS(maxprocs)
	glog.V(1).Info("Set max threads to ", maxprocs)

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

	http.Serve(cfg)
	glog.Info("Gracefully shut down")
}

func main() {
	Boot()
}
