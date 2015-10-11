// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package api implements a RESTful HTTP JSON API server for a BitTorrent
// tracker.
package api

import (
	"net"
	"net/http"
	"time"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/tylerb/graceful"

	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/stats"
	"github.com/chihaya/chihaya/tracker"
)

// Server represents an API server for a torrent tracker.
type Server struct {
	config   *config.Config
	tracker  *tracker.Tracker
	grace    *graceful.Server
	stopping bool
}

// NewServer returns a new API server for a given configuration and tracker
// instance.
func NewServer(cfg *config.Config, tkr *tracker.Tracker) *Server {
	return &Server{
		config:  cfg,
		tracker: tkr,
	}
}

// Stop cleanly shuts down the server.
func (s *Server) Stop() {
	if !s.stopping {
		s.grace.Stop(s.grace.Timeout)
	}
}

// Serve runs an API server, blocking until the server has shut down.
func (s *Server) Serve() {
	glog.V(0).Info("Starting API on ", s.config.APIConfig.ListenAddr)

	if s.config.APIConfig.ListenLimit != 0 {
		glog.V(0).Info("Limiting connections to ", s.config.APIConfig.ListenLimit)
	}

	grace := &graceful.Server{
		Timeout:     s.config.APIConfig.RequestTimeout.Duration,
		ConnState:   s.connState,
		ListenLimit: s.config.APIConfig.ListenLimit,

		NoSignalHandling: true,
		Server: &http.Server{
			Addr:         s.config.APIConfig.ListenAddr,
			Handler:      newRouter(s),
			ReadTimeout:  s.config.APIConfig.ReadTimeout.Duration,
			WriteTimeout: s.config.APIConfig.WriteTimeout.Duration,
		},
	}

	s.grace = grace
	grace.SetKeepAlivesEnabled(false)
	grace.ShutdownInitiated = func() { s.stopping = true }

	if err := grace.ListenAndServe(); err != nil {
		if opErr, ok := err.(*net.OpError); !ok || (ok && opErr.Op != "accept") {
			glog.Errorf("Failed to gracefully run API server: %s", err.Error())
			return
		}
	}

	glog.Info("API server shut down cleanly")
}

// newRouter returns a router with all the routes.
func newRouter(s *Server) *httprouter.Router {
	r := httprouter.New()

	if s.config.PrivateEnabled {
		r.PUT("/users/:passkey", makeHandler(s.putUser))
		r.DELETE("/users/:passkey", makeHandler(s.delUser))
	}

	if s.config.ClientWhitelistEnabled {
		r.GET("/clients/:clientID", makeHandler(s.getClient))
		r.PUT("/clients/:clientID", makeHandler(s.putClient))
		r.DELETE("/clients/:clientID", makeHandler(s.delClient))
	}

	r.GET("/torrents/:infohash", makeHandler(s.getTorrent))
	r.PUT("/torrents/:infohash", makeHandler(s.putTorrent))
	r.DELETE("/torrents/:infohash", makeHandler(s.delTorrent))
	r.GET("/check", makeHandler(s.check))
	r.GET("/stats", makeHandler(s.stats))

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
		glog.Errorf("Connection transitioned to unknown state %s (%d)", state, state)
	}
}

// ResponseHandler is an HTTP handler that returns a status code.
type ResponseHandler func(http.ResponseWriter, *http.Request, httprouter.Params) (int, error)

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

		if len(msg) > 0 || glog.V(2) {
			reqString := r.URL.Path + " " + r.RemoteAddr
			if glog.V(3) {
				reqString = r.URL.RequestURI() + " " + r.RemoteAddr
			}

			if len(msg) > 0 {
				glog.Errorf("[API - %9s] %s (%d - %s)", duration, reqString, httpCode, msg)
			} else {
				glog.Infof("[API - %9s] %s (%d)", duration, reqString, httpCode)
			}
		}

		stats.RecordEvent(stats.HandledRequest)
		stats.RecordTiming(stats.ResponseTime, duration)
	}
}
