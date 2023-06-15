// Package metrics implements a standalone HTTP server for serving pprof
// profiles and Prometheus metrics.
package metrics

import (
	"context"
	"errors"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/chihaya/chihaya/pkg/log"
	"github.com/chihaya/chihaya/pkg/stop"
)

// Server represents a standalone HTTP server for serving a Prometheus metrics
// endpoint.
type Server struct {
	srv *http.Server
}

// Stop shuts down the server.
func (s *Server) Stop() stop.Result {
	c := make(stop.Channel)
	go func() {
		c.Done(s.srv.Shutdown(context.Background()))
	}()

	return c.Result()
}

// NewServer creates a new instance of a Prometheus server that asynchronously
// serves requests.
func NewServer(addr string) *Server {
	mux := http.NewServeMux()

	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	s := &Server{
		srv: &http.Server{
			Addr:              addr,
			Handler:           mux,
			ReadHeaderTimeout: time.Second * 60,
		},
	}

	go func() {
		if err := s.srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("failed while serving prometheus", log.Err(err))
		}
	}()

	return s
}
