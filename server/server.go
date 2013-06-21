package server

import (
	"bytes"
	"errors"
	"net"
	"net/http"
	"path"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/jzelinskie/bufferpool"

	"github.com/jzelinskie/chihaya/config"
	"github.com/jzelinskie/chihaya/storage"
)

type Server struct {
	http.Server
	listener *net.Listener
}

func New(conf *config.Config) {
	return &Server{
		Addr:    conf.Addr,
		Handler: newHandler(conf),
	}
}

func (s *Server) Start() error {
	s.listener, err = net.Listen("tcp", config.Addr)
	if err != nil {
		return err
	}
	s.Handler.terminated = false
	s.Serve(s.listener)
	s.Handler.waitgroup.Wait()
	s.Handler.storage.Shutdown()
	return nil
}

func (s *Server) Stop() error {
	s.Handler.waitgroup.Wait()
	s.Handler.terminated = true
	return s.Handler.listener.Close()
}

type handler struct {
	bufferpool    *bufferpool.BufferPool
	conf          *config.Config
	deltaRequests int64
	storage       *storage.Storage
	terminated    bool
	waitgroup     sync.WaitGroup
}

func newHandler(conf *config.Config) {
	return &Handler{
		bufferpool: bufferpool.New(conf.BufferPoolSize, 500),
		conf:       conf,
		storage:    storage.New(&conf.Storage),
	}
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.terminated {
		return
	}

	h.waitgroup.Add(1)
	defer h.waitgroup.Done()

	if r.URL.Path == "/stats" {
		h.serveStats(&w, r)
		return
	}

	dir, action := path.Split(requestPath)
	switch action {
	case "announce":
		h.serveAnnounce(&w, r)
		return
	case "scrape":
		// TODO
		h.serveScrape(&w, r)
		return
	default:
		buf := h.bufferpool.Take()
		fail(errors.New("Unknown action"), buf)
		h.writeResponse(&w, r, buf)
		return
	}
}

func writeResponse(w *http.ResponseWriter, r *http.Request, buf *bytes.Buffer) {
	r.Close = true
	w.Header().Add("Content-Type", "text/plain")
	w.Header().Add("Connection", "close")
	w.Header().Add("Content-Length", strconv.Itoa(buf.Len()))
	w.Write(buf.Bytes())
	w.(http.Flusher).Flush()
	atomic.AddInt64(h.deltaRequests, 1)
}

func fail(err error, buf *bytes.Buffer) {
	buf.WriteString("d14:failure reason")
	buf.WriteString(strconv.Itoa(len(err)))
	buf.WriteRune(':')
	buf.WriteString(err)
	buf.WriteRune('e')
}

func validatePasskey(dir string, s *storage.Storage) (storage.User, error) {
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

func determineIP(r *http.Request, pq *parsedQuery) (string, error) {
	ip, ok := pq.params["ip"]
	if !ok {
		ip, ok = pq.params["ipv4"]
		if !ok {
			ips, ok := r.Header["X-Real-Ip"]
			if ok && len(ips) > 0 {
				ip = ips[0]
			} else {
				portIndex := len(r.RemoteAddr) - 1
				for ; portIndex >= 0; portIndex-- {
					if r.RemoteAddr[portIndex] == ':' {
						break
					}
				}
				if portIndex != -1 {
					ip = r.RemoteAddr[0:portIndex]
				} else {
					return "", errors.New("Failed to parse IP address")
				}
			}
		}
	}
	return &ip, nil
}
