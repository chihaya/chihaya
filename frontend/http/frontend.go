// Package http implements a BitTorrent frontend via the HTTP protocol as
// described in BEP 3 and BEP 23.
package http

import (
	"context"
	"crypto/tls"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/julienschmidt/httprouter"

	"github.com/chihaya/chihaya/bittorrent"
	"github.com/chihaya/chihaya/frontend"
	"github.com/chihaya/chihaya/pkg/stop"
)

// Config represents all of the configurable options for an HTTP BitTorrent
// Frontend.
type Config struct {
	Addr                string        `yaml:"addr"`
	HTTPSAddr           string        `yaml:"https_addr"`
	ReadTimeout         time.Duration `yaml:"read_timeout"`
	WriteTimeout        time.Duration `yaml:"write_timeout"`
	IdleTimeout         time.Duration `yaml:"idle_timeout"`
	EnableKeepAlive     bool          `yaml:"enable_keepalive"`
	TLSCertPath         string        `yaml:"tls_cert_path"`
	TLSKeyPath          string        `yaml:"tls_key_path"`
	AnnounceRoutes      []string      `yaml:"announce_routes"`
	ScrapeRoutes        []string      `yaml:"scrape_routes"`
	EnableRequestTiming bool          `yaml:"enable_request_timing"`
	ParseOptions        `yaml:",inline"`
}

// LogValue renders a config as a set of log fields.
func (cfg Config) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("addr", cfg.Addr),
		slog.String("httpsAddr", cfg.HTTPSAddr),
		slog.Duration("readTimeout", cfg.ReadTimeout),
		slog.Duration("writeTimeout", cfg.WriteTimeout),
		slog.Duration("idleTimeout", cfg.IdleTimeout),
		slog.Bool("enableKeepAlive", cfg.EnableKeepAlive),
		slog.String("tlsCertPath", cfg.TLSCertPath),
		slog.String("tlsKeyPath", cfg.TLSKeyPath),
		slog.Any("announceRoutes", cfg.AnnounceRoutes),
		slog.Any("scrapeRoutes", cfg.ScrapeRoutes),
		slog.Bool("enableRequestTiming", cfg.EnableRequestTiming),
		slog.Any("parseOptions", &cfg.ParseOptions),
	)
}

// Default config constants.
const (
	defaultReadTimeout  = 2 * time.Second
	defaultWriteTimeout = 2 * time.Second
	defaultIdleTimeout  = 30 * time.Second
)

// Validate sanity checks values set in a config and returns a new config with
// default values replacing anything that is invalid.
//
// This function warns to the logger when a value is changed.
func (cfg Config) Validate() Config {
	validcfg := cfg

	if cfg.ReadTimeout <= 0 {
		validcfg.ReadTimeout = defaultReadTimeout
		slog.Warn(
			"falling back to default configuration",
			slog.String("name", "http.ReadTimeout"),
			slog.Duration("provided", cfg.ReadTimeout),
			slog.Duration("default", validcfg.ReadTimeout),
		)
	}

	if cfg.WriteTimeout <= 0 {
		validcfg.WriteTimeout = defaultWriteTimeout
		slog.Warn(
			"falling back to default configuration",
			slog.String("name", "http.WriteTimeout"),
			slog.Duration("provided", cfg.WriteTimeout),
			slog.Duration("default", validcfg.WriteTimeout),
		)
	}

	if cfg.IdleTimeout <= 0 {
		validcfg.IdleTimeout = defaultIdleTimeout

		if cfg.EnableKeepAlive {
			// If keepalive is disabled, this configuration isn't used anyway.
			slog.Warn(
				"falling back to default configuration",
				slog.String("name", "http.IdleTimeout"),
				slog.Duration("provided", cfg.IdleTimeout),
				slog.Duration("default", validcfg.IdleTimeout),
			)
		}
	}

	if cfg.MaxNumWant <= 0 {
		validcfg.MaxNumWant = defaultMaxNumWant
		slog.Warn(
			"falling back to default configuration",
			slog.String("name", "http.MaxNumWant"),
			slog.Uint64("provided", uint64(cfg.MaxNumWant)),
			slog.Uint64("default", uint64(validcfg.MaxNumWant)),
		)
	}

	if cfg.DefaultNumWant <= 0 {
		validcfg.DefaultNumWant = defaultDefaultNumWant
		slog.Warn(
			"falling back to default configuration",
			slog.String("name", "http.DefaultNumWant"),
			slog.Uint64("provided", uint64(cfg.DefaultNumWant)),
			slog.Uint64("default", uint64(validcfg.DefaultNumWant)),
		)
	}

	if cfg.MaxScrapeInfoHashes <= 0 {
		validcfg.MaxScrapeInfoHashes = defaultMaxScrapeInfoHashes
		slog.Warn(
			"falling back to default configuration",
			slog.String("name", "http.MaxScrapeInfoHashes"),
			slog.Uint64("provided", uint64(cfg.MaxScrapeInfoHashes)),
			slog.Uint64("default", uint64(validcfg.MaxScrapeInfoHashes)),
		)
	}

	return validcfg
}

// Frontend represents the state of an HTTP BitTorrent Frontend.
type Frontend struct {
	srv    *http.Server
	tlsSrv *http.Server
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

	if cfg.Addr == "" && cfg.HTTPSAddr == "" {
		return nil, errors.New("must specify addr or https_addr or both")
	}

	if len(cfg.AnnounceRoutes) < 1 || len(cfg.ScrapeRoutes) < 1 {
		return nil, errors.New("must specify routes")
	}

	// If TLS is enabled, create a key pair.
	if cfg.TLSCertPath != "" && cfg.TLSKeyPath != "" {
		var err error
		f.tlsCfg = &tls.Config{
			MinVersion:   tls.VersionTLS12,
			Certificates: make([]tls.Certificate, 1),
		}
		f.tlsCfg.Certificates[0], err = tls.LoadX509KeyPair(cfg.TLSCertPath, cfg.TLSKeyPath)
		if err != nil {
			return nil, err
		}
	}

	if cfg.HTTPSAddr != "" && f.tlsCfg == nil {
		return nil, errors.New("must specify tls_cert_path and tls_key_path when using https_addr")
	}
	if cfg.HTTPSAddr == "" && f.tlsCfg != nil {
		return nil, errors.New("must specify https_addr when using tls_cert_path and tls_key_path")
	}

	var listenerHTTP, listenerHTTPS net.Listener
	var err error
	if cfg.Addr != "" {
		listenerHTTP, err = net.Listen("tcp", f.Addr)
		if err != nil {
			return nil, err
		}
	}
	if cfg.HTTPSAddr != "" {
		listenerHTTPS, err = net.Listen("tcp", f.HTTPSAddr)
		if err != nil {
			if listenerHTTP != nil {
				listenerHTTP.Close()
			}
			return nil, err
		}
	}

	if cfg.Addr != "" {
		go func() {
			if err := f.serveHTTP(listenerHTTP); err != nil {
				slog.Error("failed while serving http", slog.Any("error", err))
				os.Exit(1)
			}
		}()
	}

	if cfg.HTTPSAddr != "" {
		go func() {
			if err := f.serveHTTPS(listenerHTTPS); err != nil {
				slog.Error("failed while serving https", slog.Any("error", err))
				os.Exit(1)
			}
		}()
	}

	return f, nil
}

// Stop provides a thread-safe way to shutdown a currently running Frontend.
func (f *Frontend) Stop() stop.Result {
	stopGroup := stop.NewGroup()

	if f.srv != nil {
		stopGroup.AddFunc(f.makeStopFunc(f.srv))
	}
	if f.tlsSrv != nil {
		stopGroup.AddFunc(f.makeStopFunc(f.tlsSrv))
	}

	return stopGroup.Stop()
}

func (f *Frontend) makeStopFunc(stopSrv *http.Server) stop.Func {
	return func() stop.Result {
		c := make(stop.Channel)
		go func() {
			c.Done(stopSrv.Shutdown(context.Background()))
		}()
		return c.Result()
	}
}

func (f *Frontend) handler() http.Handler {
	router := httprouter.New()
	for _, route := range f.AnnounceRoutes {
		router.GET(route, f.announceRoute)
	}
	for _, route := range f.ScrapeRoutes {
		router.GET(route, f.scrapeRoute)
	}
	return router
}

// serveHTTP blocks while listening and serving non-TLS HTTP BitTorrent
// requests until Stop() is called or an error is returned.
func (f *Frontend) serveHTTP(l net.Listener) error {
	f.srv = &http.Server{
		Addr:         f.Addr,
		Handler:      f.handler(),
		ReadTimeout:  f.ReadTimeout,
		WriteTimeout: f.WriteTimeout,
		IdleTimeout:  f.IdleTimeout,
	}

	f.srv.SetKeepAlivesEnabled(f.EnableKeepAlive)

	// Start the HTTP server.
	if err := f.srv.Serve(l); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// serveHTTPS blocks while listening and serving TLS HTTP BitTorrent
// requests until Stop() is called or an error is returned.
func (f *Frontend) serveHTTPS(l net.Listener) error {
	f.tlsSrv = &http.Server{
		Addr:         f.HTTPSAddr,
		TLSConfig:    f.tlsCfg,
		Handler:      f.handler(),
		ReadTimeout:  f.ReadTimeout,
		WriteTimeout: f.WriteTimeout,
	}

	f.tlsSrv.SetKeepAlivesEnabled(f.EnableKeepAlive)

	// Start the HTTP server.
	if err := f.tlsSrv.ServeTLS(l, "", ""); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func injectRouteParamsToContext(ctx context.Context, ps httprouter.Params) context.Context {
	rp := make(bittorrent.RouteParams, 0, len(ps))
	for _, p := range ps {
		rp = append(rp, bittorrent.RouteParam{Key: p.Key, Value: p.Value})
	}
	return context.WithValue(ctx, bittorrent.RouteParamsKey, rp)
}

// announceRoute parses and responds to an Announce.
func (f *Frontend) announceRoute(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
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
		_ = WriteError(w, err)
		return
	}
	af = new(bittorrent.AddressFamily)
	*af = req.IP.AddressFamily

	ctx := injectRouteParamsToContext(context.Background(), ps)
	ctx, resp, err := f.logic.HandleAnnounce(ctx, req)
	if err != nil {
		_ = WriteError(w, err)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	err = WriteAnnounceResponse(w, resp)
	if err != nil {
		_ = WriteError(w, err)
		return
	}

	go f.logic.AfterAnnounce(ctx, req, resp)
}

// scrapeRoute parses and responds to a Scrape.
func (f *Frontend) scrapeRoute(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
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
		_ = WriteError(w, err)
		return
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		slog.Error(
			"http: unable to determine remote address for scrape",
			slog.Any("error", err),
		)
		_ = WriteError(w, err)
		return
	}

	reqIP := net.ParseIP(host)
	if reqIP.To4() != nil {
		req.AddressFamily = bittorrent.IPv4
	} else if len(reqIP) == net.IPv6len { // implies reqIP.To4() == nil
		req.AddressFamily = bittorrent.IPv6
	} else {
		slog.Error(
			"http: invalid IP: neither v4 nor v6",
			slog.String("remoteAddr", r.RemoteAddr),
		)
		_ = WriteError(w, bittorrent.ErrInvalidIP)
		return
	}
	af = new(bittorrent.AddressFamily)
	*af = req.AddressFamily

	ctx := injectRouteParamsToContext(context.Background(), ps)
	ctx, resp, err := f.logic.HandleScrape(ctx, req)
	if err != nil {
		_ = WriteError(w, err)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	err = WriteScrapeResponse(w, resp)
	if err != nil {
		_ = WriteError(w, err)
		return
	}

	go f.logic.AfterScrape(ctx, req, resp)
}
