// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package udp implements a UDP BitTorrent tracker per BEP 15.
// IPv6 is currently unsupported as there is no widely-implemented standard.
package udp

import (
	"net"
	"time"

	"github.com/golang/glog"
	"github.com/pushrax/bufferpool"

	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/tracker"
)

// Server represents a UDP torrent tracker.
type Server struct {
	config  *config.Config
	tracker *tracker.Tracker

	done bool
}

func (s *Server) serve() error {
	listenAddr, err := net.ResolveUDPAddr("udp", s.config.UDPListenAddr)
	if err != nil {
		return err
	}

	sock, err := net.ListenUDP("udp", listenAddr)
	defer sock.Close()
	if err != nil {
		return err
	}

	if s.config.UDPReadBufferSize > 0 {
		sock.SetReadBuffer(s.config.UDPReadBufferSize)
	}

	pool := bufferpool.New(1000, 2048)

	for !s.done {
		buffer := pool.TakeSlice()
		sock.SetReadDeadline(time.Now().Add(time.Second))
		n, addr, err := sock.ReadFromUDP(buffer)

		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
				continue
			}
			return err
		}

		start := time.Now()

		go func() {
			response, action := s.handlePacket(buffer[:n], addr)
			if response != nil {
				sock.WriteToUDP(response, addr)
			}
			pool.GiveSlice(buffer)

			if glog.V(2) {
				duration := time.Since(start)
				glog.Infof("[UDP - %9s] %s", duration, action)
			}
		}()
	}

	return nil
}

// Serve runs a UDP server, blocking until the server has shut down.
func (s *Server) Serve() {
	glog.V(0).Info("Starting UDP on ", s.config.UDPListenAddr)

	if err := s.serve(); err != nil {
		glog.Errorf("Failed to run UDP server: %s", err.Error())
	} else {
		glog.Info("UDP server shut down cleanly")
	}
}

// Stop cleanly shuts down the server.
func (s *Server) Stop() {
	s.done = true
}

// NewServer returns a new UDP server for a given configuration and tracker.
func NewServer(cfg *config.Config, tkr *tracker.Tracker) *Server {
	return &Server{
		config:  cfg,
		tracker: tkr,
	}
}
