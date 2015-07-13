// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package http implements an http-serving BitTorrent tracker.
package http

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

		if len(msg) > 0 || glog.V(2) {
			reqString := r.URL.Path + " " + r.RemoteAddr
			if glog.V(3) {
				reqString = r.URL.RequestURI() + " " + r.RemoteAddr
			}

			if len(msg) > 0 {
				glog.Errorf("[HTTP - %9s] %s (%d - %s)", duration, reqString, httpCode, msg)
			} else {
				glog.Infof("[HTTP - %9s] %s (%d)", duration, reqString, httpCode)
			}
		}

		stats.RecordEvent(stats.HandledRequest)
		stats.RecordTiming(stats.ResponseTime, duration)
	}
}

// newRouter returns a router with all the routes.
func newRouter(s *Server) *httprouter.Router {
	r := httprouter.New()

	if s.config.PrivateEnabled {
		r.GET("/users/:passkey/announce", makeHandler(s.serveAnnounce))
		r.GET("/users/:passkey/scrape", makeHandler(s.serveScrape))

		r.PUT("/users/:passkey", makeHandler(s.putUser))
		r.DELETE("/users/:passkey", makeHandler(s.delUser))
	} else {
		r.GET("/announce", makeHandler(s.serveAnnounce))
		r.GET("/scrape", makeHandler(s.serveScrape))
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

// Serve runs an HTTP server, blocking until the server has shut down.
func (s *Server) Serve(addr string) {
	glog.V(0).Info("Starting HTTP on ", addr)

	if s.config.HTTPListenLimit != 0 {
		glog.V(0).Info("Limiting connections to ", s.config.HTTPListenLimit)
	}

	grace := &graceful.Server{
		Timeout:     s.config.HTTPRequestTimeout.Duration,
		ConnState:   s.connState,
		ListenLimit: s.config.HTTPListenLimit,

		NoSignalHandling: true,
		Server: &http.Server{
			Addr:         addr,
			Handler:      newRouter(s),
			ReadTimeout:  s.config.HTTPReadTimeout.Duration,
			WriteTimeout: s.config.HTTPWriteTimeout.Duration,
		},
	}

	s.grace = grace
	grace.SetKeepAlivesEnabled(false)
	grace.ShutdownInitiated = func() { s.stopping = true }

	if err := grace.ListenAndServe(); err != nil {
		if opErr, ok := err.(*net.OpError); !ok || (ok && opErr.Op != "accept") {
			glog.Errorf("Failed to gracefully run HTTP server: %s", err.Error())
			return
		}
	}

	glog.Info("HTTP server shut down cleanly")
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
