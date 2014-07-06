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
	_ "github.com/chihaya/chihaya/drivers/backend/mock"
	_ "github.com/chihaya/chihaya/drivers/tracker/mock"
	"github.com/chihaya/chihaya/http"
)

var (
	profile    bool
	configPath string
)

func init() {
	flag.BoolVar(&profile, "profile", false, "Generate profiling data for pprof into ./chihaya.cpu")
	flag.StringVar(&configPath, "config", "", "Provide the filesystem path of a valid configuration file.")
}

func main() {
	flag.Parse()
	glog.Info("parsed flags")

	runtime.GOMAXPROCS(runtime.NumCPU())
	glog.Info("set gomaxprocs to ", runtime.NumCPU())

	if profile {
		f, err := os.Create("chihaya.cpu")
		if err != nil {
			glog.Fatalf("failed to create profile file: %s\n", err)
		}
		defer f.Close()

		pprof.StartCPUProfile(f)
		glog.Info("started profiling")

		defer func() {
			pprof.StopCPUProfile()
			glog.Info("stopped profiling")
		}()
	}

	cfg, err := config.Open(configPath)
	if err != nil {
		glog.Fatalf("failed to parse configuration file: %s\n", err)
	}
	if cfg == &config.DefaultConfig {
		glog.Info("using default config")
	} else {
		glog.Infof("loaded config file: %s", configPath)
	}

	http.Serve(cfg)
	glog.Info("gracefully shutdown")
}
