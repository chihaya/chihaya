// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package http implements a BitTorrent tracker over the HTTP protocol as per
// BEP 3.
package http

import (
	"net"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/tylerb/graceful"

	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/stats"
	"github.com/chihaya/chihaya/tracker"
	"github.com/mrd0ll4r/logger"
)

// ResponseHandler is an HTTP handler that returns a status code.
type ResponseHandler func(http.ResponseWriter, *http.Request, httprouter.Params) (int, error)

// Server represents an HTTP serving torrent tracker.
type Server struct {
	config   *config.Config
	tracker  *tracker.Tracker
	grace    *graceful.Server
	stopping bool
}

// makeHandler wraps our ResponseHandlers while timing requests, collecting,
// stats, logging, and handling errors.
func makeHandler(handler ResponseHandler) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		start := time.Now()
		httpCode, err := handler(w, r, p)
		duration := time.Since(start)

		var msg string
		if err != nil {
			msg = err.Error()
		} else if httpCode != http.StatusOK {
			msg = http.StatusText(httpCode)
		}

		if len(msg) > 0 {
			http.Error(w, msg, httpCode)
			stats.RecordEvent(stats.ErroredRequest)
		}

		if len(msg) > 0 || logger.Logs(logger.LevelInfo) {
			reqString := r.URL.Path + " " + r.RemoteAddr
			if logger.Logs(logger.LevelDebug) {
				reqString = r.URL.RequestURI() + " " + r.RemoteAddr
			}

			if len(msg) > 0 {
				logger.Warnf("[HTTP - %9s] %s (%d - %s)", duration, reqString, httpCode, msg)
			} else {
				if logger.Logs(logger.LevelDebug) {
					logger.Debugf("[HTTP - %9s] %s (%d)", duration, reqString, httpCode)
				} else {
					logger.Infof("[HTTP - %9s] %s (%d)", duration, reqString, httpCode)
				}
			}
		}

		stats.RecordEvent(stats.HandledRequest)
		stats.RecordTiming(stats.ResponseTime, duration)
	}
}

// newRouter returns a router with all the routes.
func newRouter(s *Server) *httprouter.Router {
	r := httprouter.New()

	r.GET("/announce", makeHandler(s.serveAnnounce))
	r.GET("/scrape", makeHandler(s.serveScrape))

	return r
}

// connState is used by graceful in order to gracefully shutdown. It also
// keeps track of connection stats.
func (s *Server) connState(conn net.Conn, state http.ConnState) {
	switch state {
	case http.StateNew:
		stats.RecordEvent(stats.AcceptedConnection)

	case http.StateClosed:
		stats.RecordEvent(stats.ClosedConnection)

	case http.StateHijacked:
		panic("connection impossibly hijacked")

	// Ignore the following cases.
	case http.StateActive, http.StateIdle:

	default:
		logger.Fatalf("Connection transitioned to unknown state %s (%d)", state, state)
	}
}

// Serve runs an HTTP server, blocking until the server has shut down.
func (s *Server) Serve() {
	logger.Infof("Starting HTTP on %s", s.config.HTTPConfig.ListenAddr)

	if s.config.HTTPConfig.ListenLimit != 0 {
		logger.Infof("Limiting connections to %d", s.config.HTTPConfig.ListenLimit)
	}

	grace := &graceful.Server{
		Timeout:     s.config.HTTPConfig.RequestTimeout.Duration,
		ConnState:   s.connState,
		ListenLimit: s.config.HTTPConfig.ListenLimit,

		NoSignalHandling: true,
		Server: &http.Server{
			Addr:         s.config.HTTPConfig.ListenAddr,
			Handler:      newRouter(s),
			ReadTimeout:  s.config.HTTPConfig.ReadTimeout.Duration,
			WriteTimeout: s.config.HTTPConfig.WriteTimeout.Duration,
		},
	}

	s.grace = grace
	grace.SetKeepAlivesEnabled(false)
	grace.ShutdownInitiated = func() { s.stopping = true }

	if err := grace.ListenAndServe(); err != nil {
		if opErr, ok := err.(*net.OpError); !ok || (ok && opErr.Op != "accept") {
			logger.Fatalf("Failed to gracefully run HTTP server: %s", err.Error())
			return
		}
	}

	logger.Infoln("HTTP server shut down cleanly")
}

// Stop cleanly shuts down the server.
func (s *Server) Stop() {
	if !s.stopping {
		s.grace.Stop(s.grace.Timeout)
	}
}

// NewServer returns a new HTTP server for a given configuration and tracker.
func NewServer(cfg *config.Config, tkr *tracker.Tracker) *Server {
	return &Server{
		config:  cfg,
		tracker: tkr,
	}
}
