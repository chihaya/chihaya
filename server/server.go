// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

// Package server implements a BitTorrent tracker
package server

import (
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"path"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/etix/stoppableListener"

	"github.com/chihaya/chihaya/config"
	"github.com/chihaya/chihaya/storage"
	"github.com/chihaya/chihaya/storage/backend"
	"github.com/chihaya/chihaya/storage/tracker"
)

// Server represents BitTorrent tracker server.
type Server struct {
	conf        *config.Config
	listener    *stoppableListener.StoppableListener
	trackerPool tracker.Pool
	backendConn backend.Conn

	startTime time.Time

	deltaRequests int64
	rpm           int64

	http.Server
}

// New creates a new Server.
func New(conf *config.Config) (*Server, error) {
	trackerPool, err := tracker.Open(&conf.Tracker)
	if err != nil {
		return nil, err
	}
	backendConn, err := backend.Open(&conf.Backend)
	if err != nil {
		return nil, err
	}
	err = backendConn.Start()
	if err != nil {
		return nil, err
	}

	s := &Server{
		conf:        conf,
		trackerPool: trackerPool,
		backendConn: backendConn,
		Server: http.Server{
			Addr:        conf.Addr,
			ReadTimeout: conf.ReadTimeout.Duration,
		},
	}
	s.Server.Handler = s

	return s, nil
}

// ListenAndServe starts listening and handling incoming HTTP requests.
func (s *Server) ListenAndServe() error {
	l, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	sl := stoppableListener.Handle(l)
	s.listener = sl
	s.startTime = time.Now()

	go s.updateStats()
	s.Serve(s.listener)

	return nil
}

// Stop cleanly ends the handling of incoming HTTP requests.
func (s *Server) Stop() error {
	s.listener.Stop <- true
	err := s.trackerPool.Close()
	if err != nil {
		return err
	}
	return s.listener.Close()
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer atomic.AddInt64(&s.deltaRequests, 1)
	r.Close = true

	_, action := path.Split(r.URL.Path)
	switch action {
	case "announce":
		s.serveAnnounce(w, r)
		return
	case "scrape":
		s.serveScrape(w, r)
		return
	case "stats":
		s.serveStats(w, r)
		return
	default:
		fail(errors.New("unknown action"), w, r)
		return
	}
}

func fail(err error, w http.ResponseWriter, r *http.Request) {
	errmsg := err.Error()
	message := "d14:failure reason" + strconv.Itoa(len(errmsg)) + ":" + errmsg + "e"
	length, _ := io.WriteString(w, message)
	w.Header().Add("Content-Length", string(length))
	w.(http.Flusher).Flush()
}

func validateUser(conn tracker.Conn, dir string) (*storage.User, error) {
	if len(dir) != 34 {
		return nil, errors.New("passkey is invalid")
	}
	passkey := dir[1:33]

	user, exists, err := conn.FindUser(passkey)
	if err != nil {
		log.Panicf("server: %s", err)
	}
	if !exists {
		return nil, errors.New("user not found")
	}

	return user, nil
}

// Takes a peer_id and returns a ClientID
func parsePeerID(peerID string) (clientID string) {
	length := len(peerID)
	if length >= 6 {
		if peerID[0] == '-' {
			if length >= 7 {
				clientID = peerID[1:7]
			}
		} else {
			clientID = peerID[0:6]
		}
	}
	return
}
