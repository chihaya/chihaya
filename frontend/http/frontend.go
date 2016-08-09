// Copyright 2016 Jimmy Zelinskie
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package http implements a BitTorrent frontend via the HTTP protocol as
// described in BEP 3 and BEP 23.
package http

import (
	"net"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/tylerb/graceful"
	"golang.org/x/net/context"

	"github.com/jzelinskie/trakr/frontend"
)

func init() {
	prometheus.MustRegister(promResponseDurationMilliseconds)
	recordResponseDuration("action", nil, time.Second)
}

var promResponseDurationMilliseconds = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "trakr_http_response_duration_milliseconds",
		Help:    "The duration of time it takes to receive and write a response to an API request",
		Buckets: prometheus.ExponentialBuckets(9.375, 2, 10),
	},
	[]string{"action", "error"},
)

// recordResponseDuration records the duration of time to respond to a Request
// in milliseconds .
func recordResponseDuration(action string, err error, duration time.Duration) {
	var errString string
	if err != nil {
		errString = err.Error()
	}

	promResponseDurationMilliseconds.
		WithLabelValues(action, errString).
		Observe(float64(duration.Nanoseconds()) / float64(time.Millisecond))
}

// Config represents all of the configurable options for an HTTP BitTorrent
// Frontend.
type Config struct {
	Addr            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	RequestTimeout  time.Duration
	AllowIPSpoofing bool
	RealIPHeader    string
}

// Frontend holds the state of an HTTP BitTorrent Frontend.
type Frontend struct {
	grace *graceful.Server

	backend frontend.TrackerFuncs
	Config
}

// NewFrontend allocates a new instance of a Frontend.
func NewFrontend(backend frontend.TrackerFuncs, cfg Config) *Frontend {
	return &Frontend{
		backend: backend,
		Config:  cfg,
	}
}

// Stop provides a thread-safe way to shutdown a currently running Tracker.
func (t *Frontend) Stop() {
	t.grace.Stop(t.grace.Timeout)
	<-t.grace.StopChan()
}

func (t *Frontend) handler() http.Handler {
	router := httprouter.New()
	router.GET("/announce", t.announceRoute)
	router.GET("/scrape", t.scrapeRoute)
	return router
}

// ListenAndServe listens on the TCP network address t.Addr and blocks serving
// BitTorrent requests until t.Stop() is called or an error is returned.
func (t *Frontend) ListenAndServe() error {
	t.grace = &graceful.Server{
		Server: &http.Server{
			Addr:         t.Addr,
			Handler:      t.handler(),
			ReadTimeout:  t.ReadTimeout,
			WriteTimeout: t.WriteTimeout,
		},
		Timeout:          t.RequestTimeout,
		NoSignalHandling: true,
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
	t.grace.SetKeepAlivesEnabled(false)

	if err := t.grace.ListenAndServe(); err != nil {
		if opErr, ok := err.(*net.OpError); !ok || (ok && opErr.Op != "accept") {
			panic("http: failed to gracefully run HTTP server: " + err.Error())
		}
	}

	return nil
}

// announceRoute parses and responds to an Announce by using t.TrackerFuncs.
func (t *Frontend) announceRoute(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var err error
	start := time.Now()
	defer recordResponseDuration("announce", err, time.Since(start))

	req, err := ParseAnnounce(r, t.RealIPHeader, t.AllowIPSpoofing)
	if err != nil {
		WriteError(w, err)
		return
	}

	resp, err := t.backend.HandleAnnounce(context.TODO(), req)
	if err != nil {
		WriteError(w, err)
		return
	}

	err = WriteAnnounceResponse(w, resp)
	if err != nil {
		WriteError(w, err)
		return
	}

	go t.backend.AfterAnnounce(context.TODO(), req, resp)
}

// scrapeRoute parses and responds to a Scrape by using t.TrackerFuncs.
func (t *Frontend) scrapeRoute(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var err error
	start := time.Now()
	defer recordResponseDuration("scrape", err, time.Since(start))

	req, err := ParseScrape(r)
	if err != nil {
		WriteError(w, err)
		return
	}

	resp, err := t.backend.HandleScrape(context.TODO(), req)
	if err != nil {
		WriteError(w, err)
		return
	}

	err = WriteScrapeResponse(w, resp)
	if err != nil {
		WriteError(w, err)
		return
	}

	go t.backend.AfterScrape(context.TODO(), req, resp)
}
