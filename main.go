// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"

	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/server"

	_ "github.com/chihaya/chihaya/drivers/backend/mock"
	_ "github.com/chihaya/chihaya/drivers/tracker/mock"
)

var (
	profile    bool
	configPath string
)

func init() {
	flag.BoolVar(&profile, "profile", false, "Generate profiling data for pprof into chihaya.cpu")
	flag.StringVar(&configPath, "config", "", "The location of a valid configuration file.")
}

func main() {
	flag.Parse()
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Enable the profile if flagged.
	if profile {
		log.Println("running with profiling enabled")
		f, err := os.Create("chihaya.cpu")
		if err != nil {
			log.Fatalf("failed to create profile file: %s\n", err)
		}
		defer f.Close()
		pprof.StartCPUProfile(f)
	}

	// Load the config file.
	if configPath == "" {
		log.Fatalf("must specify a configuration file")
	}
	conf, err := config.Open(configPath)
	if err != nil {
		log.Fatalf("failed to parse configuration file: %s\n", err)
	}
	log.Println("succesfully loaded config")

	// Create a new server.
	s, err := server.New(conf)
	if err != nil {
		log.Fatalf("failed to create server: %s\n", err)
	}

	// Spawn a goroutine to handle interrupts and safely shut down.
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c

		if profile {
			pprof.StopCPUProfile()
		}

		log.Println("caught interrupt, shutting down.")
		err := s.Stop()
		if err != nil {
			panic("failed to shutdown cleanly")
		}
		log.Println("shutdown successfully")
		<-c
		os.Exit(0)
	}()

	// Start the server listening and handling requests.
	err = s.ListenAndServe()
	if err != nil {
		log.Fatalf("failed to start server: %s\n", err)
	}
}
