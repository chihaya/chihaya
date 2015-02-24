// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package udp implements a UDP BitTorrent tracker per BEP 15.
// IPv6 is currently unsupported as there is no widely-implemented standard.
package udp

import (
	"errors"
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
	done    bool
	booting chan struct{}
	sock    *net.UDPConn

	connIDGen *ConnectionIDGenerator
}

func (s *Server) serve(listenAddr string) error {
	if s.sock != nil {
		return errors.New("server already booted")
	}

	udpAddr, err := net.ResolveUDPAddr("udp", listenAddr)
	if err != nil {
		close(s.booting)
		return err
	}

	sock, err := net.ListenUDP("udp", udpAddr)
	defer sock.Close()
	if err != nil {
		close(s.booting)
		return err
	}

	if s.config.UDPReadBufferSize > 0 {
		sock.SetReadBuffer(s.config.UDPReadBufferSize)
	}

	pool := bufferpool.New(1000, 2048)
	s.sock = sock
	close(s.booting)

	for !s.done {
		buffer := pool.TakeSlice()
		sock.SetReadDeadline(time.Now().Add(time.Second))
		n, addr, err := sock.ReadFromUDP(buffer)

		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
				pool.GiveSlice(buffer)
				continue
			}
			return err
		}

		start := time.Now()

		go func() {
			response, action := s.handlePacket(buffer[:n], addr)
			pool.GiveSlice(buffer)

			if len(response) > 0 {
				sock.WriteToUDP(response, addr)
			}

			if glog.V(2) {
				duration := time.Since(start)
				glog.Infof("[UDP - %9s] %s %s", duration, action, addr)
			}
		}()
	}

	return nil
}

// Serve runs a UDP server, blocking until the server has shut down.
func (s *Server) Serve(addr string) {
	glog.V(0).Info("Starting UDP on ", addr)

	go func() {
		// Generate a new IV every hour.
		for range time.Tick(time.Hour) {
			s.connIDGen.NewIV()
		}
	}()

	if err := s.serve(addr); err != nil {
		glog.Errorf("Failed to run UDP server: %s", err.Error())
	} else {
		glog.Info("UDP server shut down cleanly")
	}
}

// Stop cleanly shuts down the server.
func (s *Server) Stop() {
	s.done = true
	s.sock.SetReadDeadline(time.Now())
}

// NewServer returns a new UDP server for a given configuration and tracker.
func NewServer(cfg *config.Config, tkr *tracker.Tracker) *Server {
	gen, err := NewConnectionIDGenerator()
	if err != nil {
		panic(err)
	}

	return &Server{
		config:    cfg,
		tracker:   tkr,
		connIDGen: gen,
		booting:   make(chan struct{}),
	}
}
