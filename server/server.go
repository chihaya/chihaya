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
	"sync"
	"sync/atomic"
	"time"

	"github.com/pushrax/chihaya/config"
	"github.com/pushrax/chihaya/storage"
)

type Server struct {
	conf       *config.Config
	listener   net.Listener
	dbConnPool storage.Pool

	serving   bool
	startTime time.Time

	deltaRequests int64
	rpm           int64

	waitgroup sync.WaitGroup

	http.Server
}

func New(conf *config.Config) (*Server, error) {
	pool, err := storage.Open(&conf.Storage)
	if err != nil {
		return nil, err
	}

	s := &Server{
		conf:       conf,
		dbConnPool: pool,
		Server: http.Server{
			Addr:        conf.Addr,
			ReadTimeout: conf.ReadTimeout.Duration,
		},
	}
	s.Server.Handler = s

	return s, nil
}

func (s *Server) ListenAndServe() error {
	listener, err := net.Listen("tcp", s.Addr)
	s.listener = listener
	if err != nil {
		return err
	}
	s.serving = true
	s.startTime = time.Now()

	go s.updateRPM()
	s.Serve(s.listener)

	s.waitgroup.Wait()
	return nil
}

func (s *Server) Stop() error {
	s.serving = false
	s.waitgroup.Wait()
	err := s.dbConnPool.Close()
	if err != nil {
		return err
	}
	return s.listener.Close()
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !s.serving {
		return
	}

	s.waitgroup.Add(1)
	defer s.waitgroup.Done()
	defer atomic.AddInt64(&s.deltaRequests, 1)

	if r.URL.Path == "/stats" {
		s.serveStats(w, r)
		return
	}

	_, action := path.Split(r.URL.Path)
	switch action {
	case "announce":
		s.serveAnnounce(w, r)
		return
	case "scrape":
		s.serveScrape(w, r)
		return
	default:
		fail(errors.New("Unknown action"), w, r)
		return
	}
}

func fail(err error, w http.ResponseWriter, r *http.Request) {
	errmsg := err.Error()
	message := "d14:failure reason" + strconv.Itoa(len(errmsg)) + ":" + errmsg + "e"
	length, _ := io.WriteString(w, message)
	r.Close = true
	w.Header().Add("Content-Type", "text/plain")
	w.Header().Add("Content-Length", string(length))
	w.Header().Add("Connection", "close")
	w.(http.Flusher).Flush()
}

func validateUser(tx storage.Tx, dir string) (*storage.User, error) {
	if len(dir) != 34 {
		return nil, errors.New("Passkey is invalid")
	}
	passkey := dir[1:33]

	user, exists, err := tx.FindUser(passkey)
	if err != nil {
		log.Panicf("server: %s", err)
	}
	if !exists {
		return nil, errors.New("Passkey not found")
	}

	return user, nil
}
