/*
 * This file is part of Chihaya.
 *
 * Chihaya is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * Chihaya is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with Chihaya.  If not, see <http://www.gnu.org/licenses/>.
 */

package main

import (
	"chihaya/config"
	"chihaya/server"
	"flag"
	"log"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
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

	config.ReadConfig("config.json")

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
