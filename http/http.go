// Copyright 2014 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package http

import (
	"fmt"
	"net/http"
	"time"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/graceful"

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

type ResponseHandler func(http.ResponseWriter, *http.Request, httprouter.Params) int

func makeHandler(handler ResponseHandler) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		start := time.Now()
		code := handler(w, r, p)
		glog.Infof(
			"Completed %v %s %s in %v",
			code,
			http.StatusText(code),
			r.URL.Path,
			time.Since(start),
		)
	}
}

func setupRoutes(t *Tracker, cfg *config.Config) *httprouter.Router {
	r := httprouter.New()
	if cfg.Private {
		r.GET("/:passkey/announce", makeHandler(t.ServeAnnounce))
		r.GET("/:passkey/scrape", makeHandler(t.ServeScrape))
	} else {
		r.GET("/announce", makeHandler(t.ServeAnnounce))
		r.GET("/scrape", makeHandler(t.ServeScrape))
	}

	return r
}

func Serve(cfg *config.Config) {
	t, err := NewTracker(cfg)
	if err != nil {
		glog.Fatal("New: ", err)
	}

	graceful.Run(cfg.Addr, cfg.RequestTimeout.Duration, setupRoutes(t, cfg))
}

func fail(w http.ResponseWriter, r *http.Request, err error) {
	errmsg := err.Error()
	fmt.Fprintf(w, "d14:failure reason%d:%se", len(errmsg), errmsg)
}
