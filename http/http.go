// Copyright 2014 The Chihaya Authors. All rights reserved.
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
	"github.com/stretchr/graceful"

	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/stats"
	"github.com/chihaya/chihaya/tracker"
)

type ResponseHandler func(http.ResponseWriter, *http.Request, httprouter.Params) (int, error)

type Server struct {
	config  *config.Config
	tracker *tracker.Tracker
}

func makeHandler(handler ResponseHandler) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		var msg string

		start := time.Now()
		httpCode, err := handler(w, r, p)
		duration := time.Since(start)

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
				glog.Errorf("[%d - %9s] %s (%s)", httpCode, duration, reqString, msg)
			} else {
				glog.Infof("[%d - %9s] %s", httpCode, duration, reqString)
			}
		}

		stats.RecordEvent(stats.HandledRequest)
		stats.RecordTiming(stats.ResponseTime, duration)
	}
}

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

func (s *Server) connState(conn net.Conn, state http.ConnState) {
	switch state {
	case http.StateNew:
		stats.RecordEvent(stats.AcceptedConnection)

	case http.StateClosed:
		stats.RecordEvent(stats.ClosedConnection)

	case http.StateHijacked:
		panic("connection impossibly hijacked")

	case http.StateActive: // Ignore.
	case http.StateIdle: // Ignore.

	default:
		glog.Errorf("Connection transitioned to unknown state %s (%d)", state, state)
	}
}

func Serve(cfg *config.Config, tkr *tracker.Tracker) {
	srv := &Server{
		config:  cfg,
		tracker: tkr,
	}

	glog.V(0).Info("Starting on ", cfg.Addr)

	grace := graceful.Server{
		Timeout:   cfg.RequestTimeout.Duration,
		ConnState: srv.connState,
		Server: &http.Server{
			Addr:    cfg.Addr,
			Handler: newRouter(srv),
		},
	}

	if err := grace.ListenAndServe(); err != nil {
		glog.Errorf("Failed to start server: %s", err.Error())
	}

	if err := srv.tracker.Close(); err != nil {
		glog.Errorf("Failed to shutdown tracker cleanly: %s", err.Error())
	}
}
