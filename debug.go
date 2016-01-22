// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package chihaya

import (
	"flag"
	"net/http"
	"os"
	"runtime/pprof"

	"github.com/mrd0ll4r/logger"
	_ "net/http/pprof"
)

var (
	profile     string
	debugAddr   string
	profileFile *os.File
)

func init() {
	flag.StringVar(&profile, "profile", "", "if non-empty, path to write CPU profiling data")
	flag.StringVar(&debugAddr, "debug", "", "if non-empty, address to serve debug data")
}

func debugBoot() {
	var err error

	if debugAddr != "" {
		go func() {
			logger.Infof("Starting debug HTTP on %s", debugAddr)
			logger.Fatalln(http.ListenAndServe(debugAddr, nil))
		}()
	}

	if profile != "" {
		profileFile, err = os.Create(profile)
		if err != nil {
			logger.Fatalf("Failed to create profile file: %s", err)
		}

		pprof.StartCPUProfile(profileFile)
		logger.Infoln("Started profiling")
	}
}

func debugShutdown() {
	if profileFile != nil {
		profileFile.Close()
		pprof.StopCPUProfile()
		logger.Infoln("Stopped profiling")
	}
}
