// Package prometheus implements a standalone HTTP server for serving a
// Prometheus metrics endpoint.
package prometheus

import (
	"context"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/prometheus/client_golang/prometheus"
)

// Server represents a standalone HTTP server for serving a Prometheus metrics
// endpoint.
type Server struct {
	srv *http.Server
}

// Stop shuts down the server.
func (s *Server) Stop() <-chan error {
	c := make(chan error)
	go func() {
		if err := s.srv.Shutdown(context.Background()); err != nil {
			c <- err
		} else {
			close(c)
		}
	}()

	return c
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
			log.Fatal("failed while serving prometheus: " + err.Error())
		}
	}()

	return s
}
