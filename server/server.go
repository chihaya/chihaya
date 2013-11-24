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

  "github.com/pushrax/chihaya/config"
  "github.com/pushrax/chihaya/storage"
  "github.com/pushrax/chihaya/storage/tracker"
)

type Server struct {
  conf       *config.Config
  listener   *stoppableListener.StoppableListener
  dbConnPool tracker.Pool

  startTime time.Time

  deltaRequests int64
  rpm           int64

  http.Server
}

func New(conf *config.Config) (*Server, error) {
  pool, err := tracker.Open(&conf.Cache)
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

func (s *Server) Stop() error {
  s.listener.Stop <- true
  err := s.dbConnPool.Close()
  if err != nil {
    return err
  }
  return s.listener.Close()
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
  defer atomic.AddInt64(&s.deltaRequests, 1)
  r.Close = true

  switch r.URL.Path {
  case "/stats":
    s.serveStats(w, r)
    return
  case "/add":
    s.serveAdd(w, r)
    return
  case "/remove":
    s.serveRemove(w, r)
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
  w.Header().Add("Content-Length", string(length))
  w.(http.Flusher).Flush()
}

func validateUser(conn tracker.Conn, dir string) (*storage.User, error) {
  if len(dir) != 34 {
    return nil, errors.New("Passkey is invalid")
  }
  passkey := dir[1:33]

  user, exists, err := conn.FindUser(passkey)
  if err != nil {
    log.Panicf("server: %s", err)
  }
  if !exists {
    return nil, errors.New("User not found")
  }

  return user, nil
}

// Takes a peer_id and returns a ClientID
func parsePeerID(peerID string) (clientID string) {
  if peerID[0] == '-' {
    clientID = peerID[1:7]
  } else {
    clientID = peerID[0:6]
  }
  return
}
