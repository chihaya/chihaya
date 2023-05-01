// Package udp implements a BitTorrent tracker via the UDP protocol as
// described in BEP 15.
package udp

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/chihaya/chihaya/bittorrent"
	"github.com/chihaya/chihaya/frontend"
	"github.com/chihaya/chihaya/frontend/udp/bytepool"
	"github.com/chihaya/chihaya/pkg/log"
	"github.com/chihaya/chihaya/pkg/stop"
	"github.com/chihaya/chihaya/pkg/timecache"
)

var allowedGeneratedPrivateKeyRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")

// Config represents all of the configurable options for a UDP BitTorrent
// Tracker.
type Config struct {
	Addr                string        `yaml:"addr"`
	PrivateKey          string        `yaml:"private_key"`
	MaxClockSkew        time.Duration `yaml:"max_clock_skew"`
	EnableRequestTiming bool          `yaml:"enable_request_timing"`
	ParseOptions        `yaml:",inline"`
}

// LogFields renders the current config as a set of Logrus fields.
func (cfg Config) LogFields() log.Fields {
	return log.Fields{
		"addr":                cfg.Addr,
		"privateKey":          cfg.PrivateKey,
		"maxClockSkew":        cfg.MaxClockSkew,
		"enableRequestTiming": cfg.EnableRequestTiming,
		"allowIPSpoofing":     cfg.AllowIPSpoofing,
		"maxNumWant":          cfg.MaxNumWant,
		"defaultNumWant":      cfg.DefaultNumWant,
		"maxScrapeInfoHashes": cfg.MaxScrapeInfoHashes,
	}
}

// Validate sanity checks values set in a config and returns a new config with
// default values replacing anything that is invalid.
//
// This function warns to the logger when a value is changed.
func (cfg Config) Validate() Config {
	validcfg := cfg

	// Generate a private key if one isn't provided by the user.
	if cfg.PrivateKey == "" {
		rand.Seed(time.Now().UnixNano())
		pkeyRunes := make([]rune, 64)
		for i := range pkeyRunes {
			pkeyRunes[i] = allowedGeneratedPrivateKeyRunes[rand.Intn(len(allowedGeneratedPrivateKeyRunes))]
		}
		validcfg.PrivateKey = string(pkeyRunes)

		log.Warn("UDP private key was not provided, using generated key", log.Fields{"key": validcfg.PrivateKey})
	}

	if cfg.MaxNumWant <= 0 {
		validcfg.MaxNumWant = defaultMaxNumWant
		log.Warn("falling back to default configuration", log.Fields{
			"name":     "udp.MaxNumWant",
			"provided": cfg.MaxNumWant,
			"default":  validcfg.MaxNumWant,
		})
	}

	if cfg.DefaultNumWant <= 0 {
		validcfg.DefaultNumWant = defaultDefaultNumWant
		log.Warn("falling back to default configuration", log.Fields{
			"name":     "udp.DefaultNumWant",
			"provided": cfg.DefaultNumWant,
			"default":  validcfg.DefaultNumWant,
		})
	}

	if cfg.MaxScrapeInfoHashes <= 0 {
		validcfg.MaxScrapeInfoHashes = defaultMaxScrapeInfoHashes
		log.Warn("falling back to default configuration", log.Fields{
			"name":     "udp.MaxScrapeInfoHashes",
			"provided": cfg.MaxScrapeInfoHashes,
			"default":  validcfg.MaxScrapeInfoHashes,
		})
	}

	return validcfg
}

// Frontend holds the state of a UDP BitTorrent Frontend.
type Frontend struct {
	socket *net.UDPConn

	// A mutex for the closing channel, to make sure concurrent calls to Stop
	// are thread-safe.
	closingM sync.Mutex
	// A channel to indicate we are shutting down.
	closing chan struct{}
	// A channel that is closed once serve exits. Ensures clean shutdown.
	serveShutdown chan struct{}
	// A wait group for all goroutines spawned by the frontend.
	// After serveShutdown is closed, no more calls to Add can be made.
	wg sync.WaitGroup

	genPool *sync.Pool

	logic frontend.TrackerLogic
	Config
}

// NewFrontend creates a new instance of an UDP Frontend that asynchronously
// serves requests.
func NewFrontend(logic frontend.TrackerLogic, provided Config) (*Frontend, error) {
	cfg := provided.Validate()

	f := &Frontend{
		closing:       make(chan struct{}),
		serveShutdown: make(chan struct{}),
		logic:         logic,
		Config:        cfg,
		genPool: &sync.Pool{
			New: func() interface{} {
				return NewConnectionIDGenerator(cfg.PrivateKey)
			},
		},
	}

	if err := f.listen(); err != nil {
		return nil, err
	}

	f.wg.Add(1)
	go func() {
		defer f.wg.Done()
		if err := f.serve(); err != nil {
			log.Fatal("failed while serving udp", log.Err(err))
		}
	}()

	return f, nil
}

// Stop provides a thread-safe way to shut down a currently running Frontend.
func (t *Frontend) Stop() stop.Result {
	t.closingM.Lock()
	select {
	case <-t.closing:
		t.closingM.Unlock()
		return stop.AlreadyStopped
	default:
		close(t.closing)
	}
	t.closingM.Unlock()

	// Set socket read deadline to zero.
	// This will cancel any pending reads on the socket.
	_ = t.socket.SetReadDeadline(time.Now())

	// Depending on where the goroutine handling serve() currently is,
	// it may still launch a goroutine on previously-read data.
	// It will eventually realize we're closing and, after that, close
	// serveShutdown.
	// In order to avoid a race condition on the WaitGroup, we must ensure
	// that Add is never called after Wait. To do that, we wait for serve
	// to exit using above mechanism.
	<-t.serveShutdown

	// Finally, wait for outstanding work and close the socket.
	c := make(stop.Channel)
	go func() {
		t.wg.Wait()
		c.Done(t.socket.Close())
	}()

	return c.Result()
}

// listen resolves the address and binds the server socket.
func (t *Frontend) listen() error {
	udpAddr, err := net.ResolveUDPAddr("udp", t.Addr)
	if err != nil {
		return err
	}
	t.socket, err = net.ListenUDP("udp", udpAddr)
	return err
}

// serve blocks while listening and serving UDP BitTorrent requests
// until Stop() is called or an error is returned.
func (t *Frontend) serve() error {
	defer func() { close(t.serveShutdown) }()
	pool := bytepool.New(2048)

	for {
		// Check to see if we need to shutdown.
		select {
		case <-t.closing:
			log.Debug("udp serve() received shutdown signal")
			return nil
		default:
		}

		// Read a UDP packet into a reusable buffer.
		buffer := pool.Get()
		n, addr, err := t.socket.ReadFromUDP(*buffer)
		if err != nil {
			pool.Put(buffer)
			var netErr net.Error
			if errors.As(err, &netErr); netErr.Temporary() {
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
				Request{(*buffer)[:n], append([]byte{}, addr.IP...)},
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
	_, _ = w.socket.WriteToUDP(b, w.addr)
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

	// get a connection ID generator/validator from the pool.
	gen := t.genPool.Get().(*ConnectionIDGenerator)
	defer t.genPool.Put(gen)

	// If this isn't requesting a new connection ID and the connection ID is
	// invalid, then fail.
	if actionID != connectActionID && !gen.Validate(connID, r.IP, timecache.Now(), t.MaxClockSkew) {
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

		af = new(bittorrent.AddressFamily)
		if r.IP.To4() != nil {
			*af = bittorrent.IPv4
		} else if len(r.IP) == net.IPv6len { // implies r.IP.To4() == nil
			*af = bittorrent.IPv6
		} else {
			// Should never happen - we got the IP straight from the UDP packet.
			panic(fmt.Sprintf("udp: invalid IP: neither v4 nor v6, IP: %#v", r.IP))
		}

		WriteConnectionID(w, txID, gen.Generate(r.IP, timecache.Now()))

	case announceActionID, announceV6ActionID:
		actionName = "announce"

		var req *bittorrent.AnnounceRequest
		req, err = ParseAnnounce(r, actionID == announceV6ActionID, t.ParseOptions)
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

		WriteAnnounce(w, txID, resp, actionID == announceV6ActionID, req.IP.AddressFamily == bittorrent.IPv6)

		go t.logic.AfterAnnounce(ctx, req, resp)

	case scrapeActionID:
		actionName = "scrape"

		var req *bittorrent.ScrapeRequest
		req, err = ParseScrape(r, t.ParseOptions)
		if err != nil {
			WriteError(w, txID, err)
			return
		}

		if r.IP.To4() != nil {
			req.AddressFamily = bittorrent.IPv4
		} else if len(r.IP) == net.IPv6len { // implies r.IP.To4() == nil
			req.AddressFamily = bittorrent.IPv6
		} else {
			// Should never happen - we got the IP straight from the UDP packet.
			panic(fmt.Sprintf("udp: invalid IP: neither v4 nor v6, IP: %#v", r.IP))
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
