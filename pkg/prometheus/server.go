// Package prometheus implements a standalone HTTP server for serving a
// Prometheus metrics endpoint.
package prometheus

import (
	"context"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"

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
	s := &Server{
		srv: &http.Server{
			Addr:    addr,
			Handler: prometheus.Handler(),
		},
	}

	go func() {
		if err := s.srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal("failed while serving prometheus", log.Err(err))
		}
	}()

	return s
}
