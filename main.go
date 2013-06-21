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

	"github.com/jzelinskie/chihaya/config"
	"github.com/jzelinskie/chihaya/server"
)

var (
	profile    bool
	configFile string
)

func init() {
	flag.BoolVar(&profile, "profile", false, "Generate profiling data for pprof into chihaya.cpu")
	flag.StringVar(&configFile, "config", "", "The location of a valid configuration file.")
}

func main() {
	flag.Parse()
	runtime.GOMAXPROCS(runtime.NumCPU())

	if configFile != "" {
		conf, err := config.Parse(configFile)
		if err != nil {
			log.Fatalf("Failed to parse configuration file: %s\n", err)
		}
	}

	if profile {
		log.Println("Running with profiling enabled")
		f, err := os.Create("chihaya.cpu")
		if err != nil {
			log.Fatalf("Failed to create profile file: %s\n", err)
		}
		defer f.Close()
		pprof.StartCPUProfile(f)
	}

	s := server.New(conf)

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

	err = s.Start()
	if err != nil {
		log.Fatalf("Failed to start server: %s\n", err)
	}
}
