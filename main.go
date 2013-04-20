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

	"github.com/kotokoko/chihaya/server"
)

var profile bool

func init() {
	flag.BoolVar(&profile, "profile", false, "Generate profiling data for pprof into chihaya.cpu")
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
		pprof.StartCPUProfile(f)
	}

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c

		if profile {
			pprof.StopCPUProfile()
		}

		log.Println("Caught interrupt, shutting down..")
		server.Stop()
		<-c
		os.Exit(0)
	}()

	server.Start()
}
