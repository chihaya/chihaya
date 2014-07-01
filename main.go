// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package main

import (
	"flag"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"

	"github.com/golang/glog"

	"github.com/chihaya/chihaya/config"
	_ "github.com/chihaya/chihaya/drivers/backend/mock"
	_ "github.com/chihaya/chihaya/drivers/tracker/mock"
	"github.com/chihaya/chihaya/server"
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
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Enable the profile if flagged.
	if profile {
		f, err := os.Create("chihaya.cpu")
		if err != nil {
			glog.Fatalf("failed to create profile file: %s\n", err)
		}
		defer f.Close()

		pprof.StartCPUProfile(f)
		glog.V(1).Info("started profiling")

		defer func() {
			pprof.StopCPUProfile()
			glog.V(1).Info("stopped profiling")
		}()
	}

	// Load the config file.
	conf, err := config.Open(configPath)
	if err != nil {
		glog.Fatalf("failed to parse configuration file: %s\n", err)
	}

	// Create a new server.
	s, err := server.New(conf)
	if err != nil {
		glog.Fatalf("failed to create server: %s\n", err)
	}

	// Spawn a goroutine to handle interrupts and safely shut down.
	go func() {
		interrupts := make(chan os.Signal, 1)
		signal.Notify(interrupts, os.Interrupt)

		<-interrupts
		glog.V(1).Info("caught interrupt, shutting down...")

		err := s.Stop()
		if err != nil {
			glog.Fatalf("failed to shutdown cleanly: %s", err)
		}

		glog.V(1).Info("shutdown cleanly")

		<-interrupts

		glog.Flush()
		os.Exit(0)
	}()

	// Start the server listening and handling requests.
	err = s.ListenAndServe()
	if err != nil {
		glog.Fatalf("failed to start server: %s\n", err)
	}
}
