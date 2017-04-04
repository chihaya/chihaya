// Package http implements a BitTorrent frontend via the HTTP protocol as
// described in BEP 3 and BEP 23.
package http

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/chihaya/chihaya/bittorrent"
	"github.com/chihaya/chihaya/frontend"
)

func init() {
	prometheus.MustRegister(promResponseDurationMilliseconds)
}

// ErrInvalidIP indicates an invalid IP.
var ErrInvalidIP = bittorrent.ClientError("invalid IP")

var promResponseDurationMilliseconds = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "chihaya_http_response_duration_milliseconds",
		Help:    "The duration of time it takes to receive and write a response to an API request",
		Buckets: prometheus.ExponentialBuckets(9.375, 2, 10),
	},
	[]string{"action", "address_family", "error"},
)

// recordResponseDuration records the duration of time to respond to a Request
// in milliseconds .
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

// Config represents all of the configurable options for an HTTP BitTorrent
// Frontend.
type Config struct {
	Addr            string        `yaml:"addr"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	AllowIPSpoofing bool          `yaml:"allow_ip_spoofing"`
	RealIPHeader    string        `yaml:"real_ip_header"`
	TLSCertPath     string        `yaml:"tls_cert_path"`
	TLSKeyPath      string        `yaml:"tls_key_path"`
}

// Frontend holds the state of an HTTP BitTorrent Frontend.
type Frontend struct {
	s *http.Server

	logic frontend.TrackerLogic
	Config
}

// NewFrontend allocates a new instance of a Frontend.
func NewFrontend(logic frontend.TrackerLogic, cfg Config) *Frontend {
	return &Frontend{
		logic:  logic,
		Config: cfg,
	}
}

// Stop provides a thread-safe way to shutdown a currently running Frontend.
func (t *Frontend) Stop() {
	if err := t.s.Shutdown(context.Background()); err != nil {
		log.Warn("Error shutting down HTTP frontend:", err)
	}
}

func (t *Frontend) handler() http.Handler {
	router := httprouter.New()
	router.GET("/announce", t.announceRoute)
	router.GET("/scrape", t.scrapeRoute)
	return router
}

// ListenAndServe listens on the TCP network address t.Addr and blocks serving
// BitTorrent requests until t.Stop() is called or an error is returned.
func (t *Frontend) ListenAndServe() error {
	t.s = &http.Server{
		Addr:         t.Addr,
		Handler:      t.handler(),
		ReadTimeout:  t.ReadTimeout,
		WriteTimeout: t.WriteTimeout,
		ConnState: func(conn net.Conn, state http.ConnState) {
			switch state {
			case http.StateNew:
				//stats.RecordEvent(stats.AcceptedConnection)

			case http.StateClosed:
				//stats.RecordEvent(stats.ClosedConnection)

			case http.StateHijacked:
				panic("http: connection impossibly hijacked")

			// Ignore the following cases.
			case http.StateActive, http.StateIdle:

			default:
				panic("http: connection transitioned to unknown state")
			}
		},
	}
	t.s.SetKeepAlivesEnabled(false)

	// If TLS is enabled, create a key pair and add it to the HTTP server.
	if t.Config.TLSCertPath != "" && t.Config.TLSKeyPath != "" {
		var err error
		tlsCfg := &tls.Config{
			Certificates: make([]tls.Certificate, 1),
		}
		tlsCfg.Certificates[0], err = tls.LoadX509KeyPair(t.Config.TLSCertPath, t.Config.TLSKeyPath)
		if err != nil {
			return err
		}
		t.s.TLSConfig = tlsCfg
	}

	// Start the HTTP server and gracefully handle any network errors.
	if err := t.s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return errors.New("http: failed to run HTTP server: " + err.Error())
	}

	return nil
}

// announceRoute parses and responds to an Announce by using t.TrackerLogic.
func (t *Frontend) announceRoute(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var err error
	start := time.Now()
	var af *bittorrent.AddressFamily
	defer func() { recordResponseDuration("announce", af, err, time.Since(start)) }()

	req, err := ParseAnnounce(r, t.RealIPHeader, t.AllowIPSpoofing)
	if err != nil {
		WriteError(w, err)
		return
	}
	af = new(bittorrent.AddressFamily)
	*af = req.IP.AddressFamily

	resp, err := t.logic.HandleAnnounce(context.Background(), req)
	if err != nil {
		WriteError(w, err)
		return
	}

	err = WriteAnnounceResponse(w, resp)
	if err != nil {
		WriteError(w, err)
		return
	}

	go t.logic.AfterAnnounce(context.Background(), req, resp)
}

// scrapeRoute parses and responds to a Scrape by using t.TrackerLogic.
func (t *Frontend) scrapeRoute(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var err error
	start := time.Now()
	var af *bittorrent.AddressFamily
	defer func() { recordResponseDuration("scrape", af, err, time.Since(start)) }()

	req, err := ParseScrape(r)
	if err != nil {
		WriteError(w, err)
		return
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Errorln("http: unable to determine remote address for scrape:", err)
		WriteError(w, err)
		return
	}

	reqIP := net.ParseIP(host)
	if reqIP.To4() != nil {
		req.AddressFamily = bittorrent.IPv4
	} else if len(reqIP) == net.IPv6len { // implies reqIP.To4() == nil
		req.AddressFamily = bittorrent.IPv6
	} else {
		log.Errorln("http: invalid IP: neither v4 nor v6, RemoteAddr was", r.RemoteAddr)
		WriteError(w, ErrInvalidIP)
		return
	}
	af = new(bittorrent.AddressFamily)
	*af = req.AddressFamily

	resp, err := t.logic.HandleScrape(context.Background(), req)
	if err != nil {
		WriteError(w, err)
		return
	}

	err = WriteScrapeResponse(w, resp)
	if err != nil {
		WriteError(w, err)
		return
	}

	go t.logic.AfterScrape(context.Background(), req, resp)
}
