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

	"github.com/chihaya/bencode"
	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/drivers/backend"
	"github.com/chihaya/chihaya/drivers/tracker"
)

type Tracker struct {
	cfg *config.Config
	tp  tracker.Pool
	bc  backend.Conn
}

func NewTracker(cfg *config.Config) (*Tracker, error) {
	tp, err := tracker.Open(&cfg.Tracker)
	if err != nil {
		return nil, err
	}

	bc, err := backend.Open(&cfg.Backend)
	if err != nil {
		return nil, err
	}

	return &Tracker{
		cfg: cfg,
		tp:  tp,
		bc:  bc,
	}, nil
}

type ResponseHandler func(http.ResponseWriter, *http.Request, httprouter.Params) (int, error)

func makeHandler(handler ResponseHandler) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		start := time.Now()
		code, err := handler(w, r, p)
		if err != nil {
			http.Error(w, err.Error(), code)
		}

		glog.Infof(
			"Completed %v %s %s in %v",
			code,
			http.StatusText(code),
			r.URL.Path,
			time.Since(start),
		)
	}
}

func NewRouter(t *Tracker, cfg *config.Config) *httprouter.Router {
	r := httprouter.New()

	if cfg.Private {
		r.GET("/users/:passkey/announce", makeHandler(t.ServeAnnounce))
		r.GET("/users/:passkey/scrape", makeHandler(t.ServeScrape))

		r.PUT("/users/:passkey", makeHandler(t.putUser))
		r.DELETE("/users/:passkey", makeHandler(t.delUser))
	} else {
		r.GET("/announce", makeHandler(t.ServeAnnounce))
		r.GET("/scrape", makeHandler(t.ServeScrape))
	}

	if cfg.Whitelist {
		r.PUT("/clients/:clientID", makeHandler(t.putClient))
		r.DELETE("/clients/:clientID", makeHandler(t.delClient))
	}

	r.GET("/torrents/:infohash", makeHandler(t.getTorrent))
	r.PUT("/torrents/:infohash", makeHandler(t.putTorrent))
	r.DELETE("/torrents/:infohash", makeHandler(t.delTorrent))
	r.GET("/check", makeHandler(t.check))

	return r
}

func Serve(cfg *config.Config) {
	t, err := NewTracker(cfg)
	if err != nil {
		glog.Fatal("New: ", err)
	}

	graceful.Run(cfg.Addr, cfg.RequestTimeout.Duration, NewRouter(t, cfg))
}

func fail(w http.ResponseWriter, r *http.Request, err error) {
	dict := bencode.NewDict()
	dict["failure reason"] = err.Error()

	bencoder := bencode.NewEncoder(w)
	bencoder.Encode(dict)
}
