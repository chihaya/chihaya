// Copyright 2016 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package http

import (
	"errors"
	"log"
	"net"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/tylerb/graceful"

	"github.com/chihaya/chihaya"
	"github.com/chihaya/chihaya/server"
	"github.com/chihaya/chihaya/tracker"
)

func init() {
	server.Register("http", constructor)
}

func constructor(srvcfg *chihaya.ServerConfig, tkr *tracker.Tracker) (server.Server, error) {
	cfg, err := newHTTPConfig(srvcfg)
	if err != nil {
		return nil, errors.New("http: invalid config: " + err.Error())
	}

	return &httpServer{
		cfg: cfg,
		tkr: tkr,
	}, nil
}

type httpServer struct {
	cfg      *httpConfig
	tkr      *tracker.Tracker
	grace    *graceful.Server
	stopping bool
}

func (s *httpServer) Start() {
	s.grace = &graceful.Server{
		Server: &http.Server{
			Addr:         s.cfg.Addr,
			Handler:      s.routes(),
			ReadTimeout:  s.cfg.ReadTimeout,
			WriteTimeout: s.cfg.WriteTimeout,
		},
		Timeout:           s.cfg.RequestTimeout,
		NoSignalHandling:  true,
		ShutdownInitiated: func() { s.stopping = true },
		ConnState: func(conn net.Conn, state http.ConnState) {
			switch state {
			case http.StateNew:
				//stats.RecordEvent(stats.AcceptedConnection)

			case http.StateClosed:
				//stats.RecordEvent(stats.ClosedConnection)

			case http.StateHijacked:
				panic("http: connection impossibly hijacked")

			// Ignore the following cases.
			case http.StateActive, http.StateIdle:

			default:
				panic("http: connection transitioned to unknown state")
			}
		},
	}
	s.grace.SetKeepAlivesEnabled(false)

	if err := s.grace.ListenAndServe(); err != nil {
		if opErr, ok := err.(*net.OpError); !ok || (ok && opErr.Op != "accept") {
			log.Printf("Failed to gracefully run HTTP server: %s", err.Error())
			return
		}
	}

	log.Println("HTTP server shut down cleanly")
}

func (s *httpServer) Stop() {
	if !s.stopping {
		s.grace.Stop(s.grace.Timeout)
	}

	s.grace = nil
	s.stopping = false
}

func (s *httpServer) routes() *httprouter.Router {
	r := httprouter.New()
	r.GET("/announce", s.serveAnnounce)
	r.GET("/scrape", s.serveScrape)
	return r
}

func (s *httpServer) serveAnnounce(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	req, err := announceRequest(r, s.cfg)
	if err != nil {
		writeError(w, err)
		return
	}

	resp, err := s.tkr.HandleAnnounce(req)
	if err != nil {
		writeError(w, err)
		return
	}

	err = writeAnnounceResponse(w, resp)
	if err != nil {
		log.Println("error serializing response", err)
	}
}

func (s *httpServer) serveScrape(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	req, err := scrapeRequest(r, s.cfg)
	if err != nil {
		writeError(w, err)
		return
	}

	resp, err := s.tkr.HandleScrape(req)
	if err != nil {
		writeError(w, err)
		return
	}

	writeScrapeResponse(w, resp)
}
