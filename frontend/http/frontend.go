// Package http implements a BitTorrent frontend via the HTTP protocol as
// described in BEP 3 and BEP 23.
package http

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"

	"github.com/chihaya/chihaya/bittorrent"
	"github.com/chihaya/chihaya/frontend"
	"github.com/chihaya/chihaya/pkg/log"
)

// Config represents all of the configurable options for an HTTP BitTorrent
// Frontend.
type Config struct {
	Addr                string        `yaml:"addr"`
	ReadTimeout         time.Duration `yaml:"read_timeout"`
	WriteTimeout        time.Duration `yaml:"write_timeout"`
	TLSCertPath         string        `yaml:"tls_cert_path"`
	TLSKeyPath          string        `yaml:"tls_key_path"`
	EnableLegacyPHPURLs bool          `yaml:"enable_legacy_php_urls"`
	EnableRequestTiming bool          `yaml:"enable_request_timing"`
	ParseOptions        `yaml:",inline"`
}

// LogFields renders the current config as a set of Logrus fields.
func (cfg Config) LogFields() log.Fields {
	return log.Fields{
		"addr":                cfg.Addr,
		"readTimeout":         cfg.ReadTimeout,
		"writeTimeout":        cfg.WriteTimeout,
		"tlsCertPath":         cfg.TLSCertPath,
		"tlsKeyPath":          cfg.TLSKeyPath,
		"enableLegacyPHPURLs": cfg.EnableLegacyPHPURLs,
		"enableRequestTiming": cfg.EnableRequestTiming,
		"allowIPSpoofing":     cfg.AllowIPSpoofing,
		"realIPHeader":        cfg.RealIPHeader,
		"maxNumWant":          cfg.MaxNumWant,
		"defaultNumWant":      cfg.DefaultNumWant,
		"maxScrapeInfoHashes": cfg.MaxScrapeInfoHashes,
	}
}

// Default config constants.
const (
	defaultReadTimeout  = 2 * time.Second
	defaultWriteTimeout = 2 * time.Second
)

// Validate sanity checks values set in a config and returns a new config with
// default values replacing anything that is invalid.
//
// This function warns to the logger when a value is changed.
func (cfg Config) Validate() Config {
	validcfg := cfg

	if cfg.ReadTimeout <= 0 {
		validcfg.ReadTimeout = defaultReadTimeout
		log.Warn("falling back to default configuration", log.Fields{
			"name":     "http.ReadTimeout",
			"provided": cfg.ReadTimeout,
			"default":  validcfg.ReadTimeout,
		})
	}

	if cfg.WriteTimeout <= 0 {
		validcfg.WriteTimeout = defaultWriteTimeout
		log.Warn("falling back to default configuration", log.Fields{
			"name":     "http.WriteTimeout",
			"provided": cfg.WriteTimeout,
			"default":  validcfg.WriteTimeout,
		})
	}

	return validcfg
}

// Frontend represents the state of an HTTP BitTorrent Frontend.
type Frontend struct {
	srv    *http.Server
	tlsCfg *tls.Config

	logic frontend.TrackerLogic
	Config
}

// NewFrontend creates a new instance of an HTTP Frontend that asynchronously
// serves requests.
func NewFrontend(logic frontend.TrackerLogic, provided Config) (*Frontend, error) {
	cfg := provided.Validate()

	f := &Frontend{
		logic:  logic,
		Config: cfg,
	}

	// If TLS is enabled, create a key pair.
	if cfg.TLSCertPath != "" && cfg.TLSKeyPath != "" {
		var err error
		f.tlsCfg = &tls.Config{
			Certificates: make([]tls.Certificate, 1),
		}
		f.tlsCfg.Certificates[0], err = tls.LoadX509KeyPair(cfg.TLSCertPath, cfg.TLSKeyPath)
		if err != nil {
			return nil, err
		}
	}

	go func() {
		if err := f.listenAndServe(); err != nil {
			log.Fatal("failed while serving http", log.Err(err))
		}
	}()

	return f, nil
}

// Stop provides a thread-safe way to shutdown a currently running Frontend.
func (f *Frontend) Stop() <-chan error {
	c := make(chan error)
	go func() {
		if err := f.srv.Shutdown(context.Background()); err != nil {
			c <- err
		} else {
			close(c)
		}
	}()

	return c
}

func (f *Frontend) handler() http.Handler {
	router := httprouter.New()
	router.GET("/announce", f.announceRoute)
	router.GET("/scrape", f.scrapeRoute)

	if f.EnableLegacyPHPURLs {
		log.Info("http: enabling legacy PHP URLs")
		router.GET("/announce.php", f.announceRoute)
		router.GET("/scrape.php", f.scrapeRoute)
	}

	return router
}

// listenAndServe blocks while listening and serving HTTP BitTorrent requests
// until Stop() is called or an error is returned.
func (f *Frontend) listenAndServe() error {
	f.srv = &http.Server{
		Addr:         f.Addr,
		TLSConfig:    f.tlsCfg,
		Handler:      f.handler(),
		ReadTimeout:  f.ReadTimeout,
		WriteTimeout: f.WriteTimeout,
	}

	// Disable KeepAlives.
	f.srv.SetKeepAlivesEnabled(false)

	// Start the HTTP server.
	if err := f.srv.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}

	return nil
}

// announceRoute parses and responds to an Announce.
func (f *Frontend) announceRoute(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var err error
	var start time.Time
	if f.EnableRequestTiming {
		start = time.Now()
	}
	var af *bittorrent.AddressFamily
	defer func() {
		if f.EnableRequestTiming {
			recordResponseDuration("announce", af, err, time.Since(start))
		} else {
			recordResponseDuration("announce", af, err, time.Duration(0))
		}
	}()

	req, err := ParseAnnounce(r, f.ParseOptions)
	if err != nil {
		WriteError(w, err)
		return
	}
	af = new(bittorrent.AddressFamily)
	*af = req.IP.AddressFamily

	ctx, resp, err := f.logic.HandleAnnounce(context.Background(), req)
	if err != nil {
		WriteError(w, err)
		return
	}

	err = WriteAnnounceResponse(w, resp)
	if err != nil {
		WriteError(w, err)
		return
	}

	go f.logic.AfterAnnounce(ctx, req, resp)
}

// scrapeRoute parses and responds to a Scrape.
func (f *Frontend) scrapeRoute(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var err error
	var start time.Time
	if f.EnableRequestTiming {
		start = time.Now()
	}
	var af *bittorrent.AddressFamily
	defer func() {
		if f.EnableRequestTiming {
			recordResponseDuration("scrape", af, err, time.Since(start))
		} else {
			recordResponseDuration("scrape", af, err, time.Duration(0))
		}
	}()

	req, err := ParseScrape(r, f.ParseOptions)
	if err != nil {
		WriteError(w, err)
		return
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Error("http: unable to determine remote address for scrape", log.Err(err))
		WriteError(w, err)
		return
	}

	reqIP := net.ParseIP(host)
	if reqIP.To4() != nil {
		req.AddressFamily = bittorrent.IPv4
	} else if len(reqIP) == net.IPv6len { // implies reqIP.To4() == nil
		req.AddressFamily = bittorrent.IPv6
	} else {
		log.Error("http: invalid IP: neither v4 nor v6", log.Fields{"RemoteAddr": r.RemoteAddr})
		WriteError(w, bittorrent.ErrInvalidIP)
		return
	}
	af = new(bittorrent.AddressFamily)
	*af = req.AddressFamily

	ctx, resp, err := f.logic.HandleScrape(context.Background(), req)
	if err != nil {
		WriteError(w, err)
		return
	}

	err = WriteScrapeResponse(w, resp)
	if err != nil {
		WriteError(w, err)
		return
	}

	go f.logic.AfterScrape(ctx, req, resp)
}
