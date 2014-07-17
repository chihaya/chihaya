// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package http implements an http-serving BitTorrent tracker.
package http

import (
	"net/http"
	"time"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/graceful"

	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/tracker"
)

type ResponseHandler func(http.ResponseWriter, *http.Request, httprouter.Params) (int, error)

type Server struct {
	config  *config.Config
	tracker *tracker.Tracker
}

func makeHandler(handler ResponseHandler) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		start := time.Now()

		httpCode, err := handler(w, r, p)
		if err != nil {
			http.Error(w, err.Error(), httpCode)
		}

		if glog.V(2) {
			glog.Infof(
				"Completed %v %s %s in %v",
				httpCode,
				http.StatusText(httpCode),
				r.URL.Path,
				time.Since(start),
			)
		}
	}
}

func NewRouter(s *Server) *httprouter.Router {
	r := httprouter.New()

	if s.config.Private {
		r.GET("/users/:passkey/announce", makeHandler(s.serveAnnounce))
		r.GET("/users/:passkey/scrape", makeHandler(s.serveScrape))

		r.PUT("/users/:passkey", makeHandler(s.putUser))
		r.DELETE("/users/:passkey", makeHandler(s.delUser))
	} else {
		r.GET("/announce", makeHandler(s.serveAnnounce))
		r.GET("/scrape", makeHandler(s.serveScrape))
	}

	if s.config.Whitelist {
		r.PUT("/clients/:clientID", makeHandler(s.putClient))
		r.DELETE("/clients/:clientID", makeHandler(s.delClient))
	}

	r.GET("/torrents/:infohash", makeHandler(s.getTorrent))
	r.PUT("/torrents/:infohash", makeHandler(s.putTorrent))
	r.DELETE("/torrents/:infohash", makeHandler(s.delTorrent))
	r.GET("/check", makeHandler(s.check))

	return r
}

func Serve(cfg *config.Config, tkr *tracker.Tracker) {
	srv := &Server{
		config:  cfg,
		tracker: tkr,
	}

	glog.V(0).Info("Starting on ", cfg.Addr)
	graceful.Run(cfg.Addr, cfg.RequestTimeout.Duration, NewRouter(srv))
}
