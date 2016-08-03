// Copyright 2016 Jimmy Zelinskie
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package udp

import (
	"bytes"
	"encoding/binary"
	"net"
	"time"

	"github.com/jzelinskie/trakr/bittorrent"
)

var promResponseDurationMilliseconds = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "trakr_udp_response_duration_milliseconds",
		Help:    "The duration of time it takes to receive and write a response to an API request",
		Buckets: prometheus.ExponentialBuckets(9.375, 2, 10),
	},
	[]string{"action", "error"},
)

type Config struct {
	Addr            string
	PrivateKey      string
	AllowIPSpoofing bool
}

type Server struct {
	sock    *net.UDPConn
	closing chan struct{}
	wg      sync.WaitGroup

	bittorrent.ServerFuncs
	Config
}

func NewServer(funcs bittorrent.ServerFuncs, cfg Config) {
	return &Server{
		closing:     make(chan struct{}),
		ServerFuncs: funcs,
		Config:      cfg,
	}
}

func (s *udpServer) Stop() {
	close(s.closing)
	s.sock.SetReadDeadline(time.Now())
	s.wg.Wait()
}

func (s *Server) ListenAndServe() error {
	udpAddr, err := net.ResolveUDPAddr("udp", s.Addr)
	if err != nil {
		return err
	}

	s.sock, err = net.ListenUDP("udp", udpAddr)
	if err != nil {
		return err
	}
	defer s.sock.Close()

	pool := bytepool.New(256, 2048)

	for {
		// Check to see if we need to shutdown.
		select {
		case <-s.closing:
			s.wg.Wait()
			return nil
		default:
		}

		// Read a UDP packet into a reusable buffer.
		buffer := pool.Get()
		s.sock.SetReadDeadline(time.Now().Add(time.Second))
		n, addr, err := s.sock.ReadFromUDP(buffer)
		if err != nil {
			pool.Put(buffer)
			if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
				// A temporary failure is not fatal; just pretend it never happened.
				continue
			}
			return err
		}

		// We got nothin'
		if n == 0 {
			pool.Put(buffer)
			continue
		}

		log.Println("Got UDP packet")
		start := time.Now()
		s.wg.Add(1)
		go func(start time.Time) {
			defer s.wg.Done()
			defer pool.Put(buffer)

			// Handle the response.
			response, action, err := s.handlePacket(buffer[:n], addr)
			log.Printf("Handled UDP packet: %s, %s, %s\n", response, action, err)

			// Record to Prometheus the time in milliseconds to receive, handle, and
			// respond to the request.
			duration := time.Since(start)
			if err != nil {
				promResponseDurationMilliseconds.WithLabelValues(action, err.Error()).Observe(float64(duration.Nanoseconds()) / float64(time.Millisecond))
			} else {
				promResponseDurationMilliseconds.WithLabelValues(action, "").Observe(float64(duration.Nanoseconds()) / float64(time.Millisecond))
			}
		}(start)
	}
}

type Request struct {
	Packet []byte
	IP     net.IP
}

type ResponseWriter struct {
	socket net.UDPConn
	addr   net.UDPAddr
}

func (w *ResponseWriter) Write(b []byte) (int, error) {
	w.socket.WriteToUDP(b, w.addr)
	return len(b), nil
}

func (s *Server) handlePacket(r *Request, w *ResponseWriter) (response []byte, actionName string, err error) {
	if len(r.packet) < 16 {
		// Malformed, no client packets are less than 16 bytes.
		// We explicitly return nothing in case this is a DoS attempt.
		err = errMalformedPacket
		return
	}

	// Parse the headers of the UDP packet.
	connID := r.packet[0:8]
	actionID := binary.BigEndian.Uint32(r.packet[8:12])
	txID := r.packet[12:16]

	// If this isn't requesting a new connection ID and the connection ID is
	// invalid, then fail.
	if actionID != connectActionID && !ValidConnectionID(connID, r.IP, time.Now(), s.PrivateKey) {
		err = errBadConnectionID
		WriteError(w, txID, err)
		return
	}

	// Handle the requested action.
	switch actionID {
	case connectActionID:
		actionName = "connect"

		if !bytes.Equal(connID, initialConnectionID) {
			err = errMalformedPacket
			return
		}

		WriteConnectionID(w, txID, NewConnectionID(r.IP, time.Now(), s.PrivateKey))
		return

	case announceActionID:
		actionName = "announce"

		var req *bittorrent.AnnounceRequest
		req, err = ParseAnnounce(r, s.AllowIPSpoofing)
		if err != nil {
			WriteError(w, txID, err)
			return
		}

		var resp *bittorrent.AnnounceResponse
		resp, err = s.HandleAnnounce(req)
		if err != nil {
			WriteError(w, txID, err)
			return
		}

		WriteAnnounce(w, txID, resp)

		if s.AfterAnnounce != nil {
			s.AfterAnnounce(req, resp)
		}

		return

	case scrapeActionID:
		actionName = "scrape"

		var req *bittorrent.ScrapeRequest
		req, err = ParseScrape(r)
		if err != nil {
			WriteError(w, txID, err)
			return
		}

		var resp *bittorrent.ScrapeResponse
		ctx := context.TODO()
		resp, err = s.HandleScrape(ctx, req)
		if err != nil {
			WriteError(w, txID, err)
			return
		}

		WriteScrape(w, txID, resp)

		if s.AfterScrape != nil {
			s.AfterScrape(req, resp)
		}

		return

	default:
		err = errUnknownAction
		WriteError(w, txID, err)
		return
	}
}
