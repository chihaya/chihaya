// Copyright 2015 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package udp implements a UDP BitTorrent tracker per BEP 15 and BEP 41.
// IPv6 is currently unsupported as there is no widely-implemented standard.
package udp

import (
	"net"

	"github.com/golang/glog"
	"github.com/pushrax/bufferpool"

	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/tracker"
)

// Server represents a UDP torrent tracker.
type Server struct {
	config  *config.Config
	tracker *tracker.Tracker
}

func (srv *Server) ListenAndServe() error {
	listenAddr, err := net.ResolveUDPAddr("udp", srv.config.UDPListenAddr)
	if err != nil {
		return err
	}

	sock, err := net.ListenUDP("udp", listenAddr)
	defer sock.Close()
	if err != nil {
		return err
	}

	if srv.config.UDPReadBufferSize > 0 {
		sock.SetReadBuffer(srv.config.UDPReadBufferSize)
	}

	pool := bufferpool.New(1000, 2048)

	for {
		buffer := pool.TakeSlice()
		n, addr, err := sock.ReadFromUDP(buffer)
		if err != nil {
			return err
		}

		go func() {
			response := srv.handlePacket(buffer[:n], addr)
			if response != nil {
				sock.WriteToUDP(response, addr)
			}
			pool.GiveSlice(buffer)
		}()
	}
}

func Serve(cfg *config.Config, tkr *tracker.Tracker) {
	srv := &Server{
		config:  cfg,
		tracker: tkr,
	}

	glog.V(0).Info("Starting UDP on ", cfg.UDPListenAddr)
	if err := srv.ListenAndServe(); err != nil {
		glog.Errorf("Failed to run UDP server: %s", err.Error())
	}
}
