// Copyright 2013 The Chihaya Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package server

import (
	"errors"
	"fmt"
	"io"
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
	conf     *config.Config
	listener net.Listener
	storage  storage.Storage

	serving   bool
	startTime time.Time

	deltaRequests int64
	rpm           int64

	waitgroup sync.WaitGroup

	http.Server
}

func New(conf *config.Config) (*Server, error) {
	store, err := storage.New(&conf.Storage)
	if err != nil {
		return nil, err
	}

	s := &Server{
		conf:    conf,
		storage: store,
		Server: http.Server{
			Addr: conf.Addr,
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
	err := s.storage.Close()
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
	defer finalizeResponse(w, r)

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

func finalizeResponse(w http.ResponseWriter, r *http.Request) {
	r.Close = true
	w.Header().Add("Content-Type", "text/plain")
	w.Header().Add("Connection", "close")
	w.(http.Flusher).Flush()
}

func fail(err error, w http.ResponseWriter, r *http.Request) {
	errmsg := err.Error()
	message := fmt.Sprintf(
		"%s%s%s%s%s",
		"d14:failure reason",
		strconv.Itoa(len(errmsg)),
		":",
		errmsg,
		"e",
	)
	io.WriteString(w, message)
}

func validatePasskey(dir string, s storage.Storage) (*storage.User, error) {
	if len(dir) != 34 {
		return nil, errors.New("Your passkey is invalid")
	}
	passkey := dir[1:33]

	user, exists, err := s.FindUser(passkey)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("Passkey not found")
	}

	return user, nil
}
