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

	"github.com/pushrax/chihaya/config"
	"github.com/pushrax/chihaya/server"

	_ "github.com/pushrax/chihaya/storage/tracker/redis"
	_ "github.com/pushrax/chihaya/storage/web/batter"
	_ "github.com/pushrax/chihaya/storage/web/gazelle"
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

	if profile {
		log.Println("Running with profiling enabled")
		f, err := os.Create("chihaya.cpu")
		if err != nil {
			log.Fatalf("Failed to create profile file: %s\n", err)
		}
		defer f.Close()
		pprof.StartCPUProfile(f)
	}

	if configPath == "" {
		log.Fatalf("Must specify a configuration file")
	}
	conf, err := config.Open(configPath)
	if err != nil {
		log.Fatalf("Failed to parse configuration file: %s\n", err)
	}
	s, err := server.New(conf)
	if err != nil {
		log.Fatalf("Failed to create server: %s\n", err)
	}

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c

		if profile {
			pprof.StopCPUProfile()
		}

		log.Println("Caught interrupt, shutting down..")
		err := s.Stop()
		if err != nil {
			panic("Failed to shutdown cleanly")
		}
		log.Println("Shutdown successfully")
		<-c
		os.Exit(0)
	}()

	err = s.ListenAndServe()
	if err != nil {
		log.Fatalf("Failed to start server: %s\n", err)
	}
}
