// Package udp implements a BitTorrent tracker via the UDP protocol as
// described in BEP 15.
package udp

import (
	"bytes"
	"context"
	"encoding/binary"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/chihaya/chihaya/bittorrent"
	"github.com/chihaya/chihaya/frontend"
	"github.com/chihaya/chihaya/frontend/udp/bytepool"
	"github.com/chihaya/chihaya/pkg/log"
	"github.com/chihaya/chihaya/pkg/stop"
	"github.com/chihaya/chihaya/pkg/timecache"
)

var allowedGeneratedPrivateKeyRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")

func init() {
	prometheus.MustRegister(promResponseDurationMilliseconds)
}

// ErrInvalidIP indicates an invalid IP.
var ErrInvalidIP = bittorrent.ClientError("invalid IP")

var promResponseDurationMilliseconds = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "chihaya_udp_response_duration_milliseconds",
		Help:    "The duration of time it takes to receive and write a response to an API request",
		Buckets: prometheus.ExponentialBuckets(9.375, 2, 10),
	},
	[]string{"action", "address_family", "error"},
)

// recordResponseDuration records the duration of time to respond to a UDP
// Request in milliseconds .
func recordResponseDuration(action string, af *bittorrent.AddressFamily, err error, duration time.Duration) {
	var errString string
	if err != nil {
		if _, ok := err.(bittorrent.ClientError); ok {
			errString = err.Error()
		} else {
			errString = "internal error"
		}
	}

	var afString string
	if af == nil {
		afString = "Unknown"
	} else if *af == bittorrent.IPv4 {
		afString = "IPv4"
	} else if *af == bittorrent.IPv6 {
		afString = "IPv6"
	}

	promResponseDurationMilliseconds.
		WithLabelValues(action, afString, errString).
		Observe(float64(duration.Nanoseconds()) / float64(time.Millisecond))
}

// Config represents all of the configurable options for a UDP BitTorrent
// Tracker.
type Config struct {
	Addr                string        `yaml:"addr"`
	PrivateKey          string        `yaml:"private_key"`
	MaxClockSkew        time.Duration `yaml:"max_clock_skew"`
	AllowIPSpoofing     bool          `yaml:"allow_ip_spoofing"`
	EnableRequestTiming bool          `yaml:"enable_request_timing"`
}

// LogFields renders the current config as a set of Logrus fields.
func (cfg Config) LogFields() log.Fields {
	return log.Fields{
		"addr":                cfg.Addr,
		"privateKey":          cfg.PrivateKey,
		"maxClockSkew":        cfg.MaxClockSkew,
		"allowIPSpoofing":     cfg.AllowIPSpoofing,
		"enableRequestTiming": cfg.EnableRequestTiming,
	}
}

// Frontend holds the state of a UDP BitTorrent Frontend.
type Frontend struct {
	socket  *net.UDPConn
	closing chan struct{}
	wg      sync.WaitGroup

	logic frontend.TrackerLogic
	Config
}

// NewFrontend creates a new instance of an UDP Frontend that asynchronously
// serves requests.
func NewFrontend(logic frontend.TrackerLogic, cfg Config) (*Frontend, error) {
	// Generate a private key if one isn't provided by the user.
	if cfg.PrivateKey == "" {
		rand.Seed(time.Now().UnixNano())
		pkeyRunes := make([]rune, 64)
		for i := range pkeyRunes {
			pkeyRunes[i] = allowedGeneratedPrivateKeyRunes[rand.Intn(len(allowedGeneratedPrivateKeyRunes))]
		}
		cfg.PrivateKey = string(pkeyRunes)

		log.Warn("UDP private key was not provided, using generated key", log.Fields{"key": cfg.PrivateKey})
	}

	f := &Frontend{
		closing: make(chan struct{}),
		logic:   logic,
		Config:  cfg,
	}

	go func() {
		if err := f.listenAndServe(); err != nil {
			log.Fatal("failed while serving udp", log.Err(err))
		}
	}()

	return f, nil
}

// Stop provides a thread-safe way to shutdown a currently running Frontend.
func (t *Frontend) Stop() <-chan error {
	select {
	case <-t.closing:
		return stop.AlreadyStopped
	default:
	}

	c := make(chan error)
	go func() {
		close(t.closing)
		t.socket.SetReadDeadline(time.Now())
		t.wg.Wait()
		if err := t.socket.Close(); err != nil {
			c <- err
		} else {
			close(c)
		}
	}()

	return c
}

// listenAndServe blocks while listening and serving UDP BitTorrent requests
// until Stop() is called or an error is returned.
func (t *Frontend) listenAndServe() error {
	udpAddr, err := net.ResolveUDPAddr("udp", t.Addr)
	if err != nil {
		return err
	}

	t.socket, err = net.ListenUDP("udp", udpAddr)
	if err != nil {
		return err
	}

	pool := bytepool.New(2048)

	t.wg.Add(1)
	defer t.wg.Done()

	for {
		// Check to see if we need to shutdown.
		select {
		case <-t.closing:
			log.Debug("udp listenAndServe() received shutdown signal")
			return nil
		default:
		}

		// Read a UDP packet into a reusable buffer.
		buffer := pool.Get()
		n, addr, err := t.socket.ReadFromUDP(buffer)
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

		t.wg.Add(1)
		go func() {
			defer t.wg.Done()
			defer pool.Put(buffer)

			if ip := addr.IP.To4(); ip != nil {
				addr.IP = ip
			}

			// Handle the request.
			var start time.Time
			if t.EnableRequestTiming {
				start = time.Now()
			}
			action, af, err := t.handleRequest(
				// Make sure the IP is copied, not referenced.
				Request{buffer[:n], append([]byte{}, addr.IP...)},
				ResponseWriter{t.socket, addr},
			)
			if t.EnableRequestTiming {
				recordResponseDuration(action, af, err, time.Since(start))
			} else {
				recordResponseDuration(action, af, err, time.Duration(0))
			}
		}()
	}
}

// Request represents a UDP payload received by a Tracker.
type Request struct {
	Packet []byte
	IP     net.IP
}

// ResponseWriter implements the ability to respond to a Request via the
// io.Writer interface.
type ResponseWriter struct {
	socket *net.UDPConn
	addr   *net.UDPAddr
}

// Write implements the io.Writer interface for a ResponseWriter.
func (w ResponseWriter) Write(b []byte) (int, error) {
	w.socket.WriteToUDP(b, w.addr)
	return len(b), nil
}

// handleRequest parses and responds to a UDP Request.
func (t *Frontend) handleRequest(r Request, w ResponseWriter) (actionName string, af *bittorrent.AddressFamily, err error) {
	if len(r.Packet) < 16 {
		// Malformed, no client packets are less than 16 bytes.
		// We explicitly return nothing in case this is a DoS attempt.
		err = errMalformedPacket
		return
	}

	// Parse the headers of the UDP packet.
	connID := r.Packet[0:8]
	actionID := binary.BigEndian.Uint32(r.Packet[8:12])
	txID := r.Packet[12:16]

	// If this isn't requesting a new connection ID and the connection ID is
	// invalid, then fail.
	if actionID != connectActionID && !ValidConnectionID(connID, r.IP, timecache.Now(), t.MaxClockSkew, t.PrivateKey) {
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

		WriteConnectionID(w, txID, NewConnectionID(r.IP, timecache.Now(), t.PrivateKey))

	case announceActionID, announceV6ActionID:
		actionName = "announce"

		var req *bittorrent.AnnounceRequest
		req, err = ParseAnnounce(r, t.AllowIPSpoofing, actionID == announceV6ActionID)
		if err != nil {
			WriteError(w, txID, err)
			return
		}
		af = new(bittorrent.AddressFamily)
		*af = req.IP.AddressFamily

		var ctx context.Context
		var resp *bittorrent.AnnounceResponse
		ctx, resp, err = t.logic.HandleAnnounce(context.Background(), req)
		if err != nil {
			WriteError(w, txID, err)
			return
		}

		WriteAnnounce(w, txID, resp, actionID == announceV6ActionID)

		go t.logic.AfterAnnounce(ctx, req, resp)

	case scrapeActionID:
		actionName = "scrape"

		var req *bittorrent.ScrapeRequest
		req, err = ParseScrape(r)
		if err != nil {
			WriteError(w, txID, err)
			return
		}

		if r.IP.To4() != nil {
			req.AddressFamily = bittorrent.IPv4
		} else if len(r.IP) == net.IPv6len { // implies r.IP.To4() == nil
			req.AddressFamily = bittorrent.IPv6
		} else {
			log.Error("udp: invalid IP: neither v4 nor v6", log.Fields{"IP": r.IP})
			WriteError(w, txID, ErrInvalidIP)
			return
		}
		af = new(bittorrent.AddressFamily)
		*af = req.AddressFamily

		var ctx context.Context
		var resp *bittorrent.ScrapeResponse
		ctx, resp, err = t.logic.HandleScrape(context.Background(), req)
		if err != nil {
			WriteError(w, txID, err)
			return
		}

		WriteScrape(w, txID, resp)

		go t.logic.AfterScrape(ctx, req, resp)

	default:
		err = errUnknownAction
		WriteError(w, txID, err)
	}

	return
}
