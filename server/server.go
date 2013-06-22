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

	"github.com/pushrax/chihaya/config"
	"github.com/pushrax/chihaya/storage"
)

type Server struct {
	conf       *config.Config
	listener   net.Listener
	storage    storage.Storage
	terminated *bool
	waitgroup  *sync.WaitGroup

	http.Server
}

func New(conf *config.Config) (*Server, error) {
	var (
		wg         sync.WaitGroup
		terminated bool
	)

	store, err := storage.New(&conf.Storage)
	if err != nil {
		return nil, err
	}

	handler := &handler{
		conf:       conf,
		storage:    store,
		terminated: &terminated,
		waitgroup:  &wg,
	}

	s := &Server{
		conf:       conf,
		storage:    store,
		terminated: &terminated,
		waitgroup:  &wg,
	}

	s.Server.Addr = conf.Addr
	s.Server.Handler = handler
	return s, nil
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.conf.Addr)
	if err != nil {
		return err
	}
	*s.terminated = false
	s.Serve(s.listener)
	s.waitgroup.Wait()
	return nil
}

func (s *Server) Stop() error {
	*s.terminated = true
	s.waitgroup.Wait()
	err := s.storage.Shutdown()
	if err != nil {
		return err
	}
	return s.listener.Close()
}

type handler struct {
	conf          *config.Config
	deltaRequests int64
	storage       storage.Storage
	terminated    *bool
	waitgroup     *sync.WaitGroup
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if *h.terminated {
		return
	}

	h.waitgroup.Add(1)
	defer h.waitgroup.Done()

	if r.URL.Path == "/stats" {
		h.serveStats(w, r)
		return
	}

	passkey, action := path.Split(r.URL.Path)
	switch action {
	case "announce":
		h.serveAnnounce(w, r)
		return
	case "scrape":
		// TODO
		h.serveScrape(w, r)
		return
	default:
		written := fail(errors.New("Unknown action"), w)
		h.finalizeResponse(w, r, written)
		return
	}
}

func (h *handler) finalizeResponse(
	w http.ResponseWriter,
	r *http.Request,
	written int,
) {
	r.Close = true
	w.Header().Add("Content-Type", "text/plain")
	w.Header().Add("Connection", "close")
	w.Header().Add("Content-Length", strconv.Itoa(written))
	w.(http.Flusher).Flush()
	atomic.AddInt64(&h.deltaRequests, 1)
}

func fail(err error, w http.ResponseWriter) int {
	e := err.Error()
	message := fmt.Sprintf(
		"%s%s%s%s%s",
		"d14:failure reason",
		strconv.Itoa(len(e)),
		':',
		e,
		'e',
	)
	written, _ := io.WriteString(w, message)
	return written
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
