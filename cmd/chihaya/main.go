package main

import (
	"errors"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jzelinskie/cobrautil"
	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/chihaya/chihaya/frontend/http"
	"github.com/chihaya/chihaya/frontend/udp"
	"github.com/chihaya/chihaya/middleware"
	"github.com/chihaya/chihaya/pkg/metrics"
	"github.com/chihaya/chihaya/pkg/stop"
	"github.com/chihaya/chihaya/storage"
)

// Run represents the state of a running instance of Chihaya.
type Run struct {
	configFilePath string
	peerStore      storage.PeerStore
	logic          *middleware.Logic
	sg             *stop.Group
}

// NewRun runs an instance of Chihaya.
func NewRun(configFilePath string) (*Run, error) {
	r := &Run{
		configFilePath: configFilePath,
	}

	return r, r.Start(nil)
}

// Start begins an instance of Chihaya.
// It is optional to provide an instance of the peer store to avoid the
// creation of a new one.
func (r *Run) Start(ps storage.PeerStore) error {
	configFile, err := ParseConfigFile(r.configFilePath)
	if err != nil {
		return errors.New("failed to read config: " + err.Error())
	}
	cfg := configFile.Chihaya

	r.sg = stop.NewGroup()

	log.Info().Str("addr", cfg.MetricsAddr).Msg("starting metrics server")
	r.sg.Add(metrics.NewServer(cfg.MetricsAddr))

	if ps == nil {
		log.Info().Str("name", cfg.Storage.Name).Msg("starting storage")
		ps, err = storage.NewPeerStore(cfg.Storage.Name, cfg.Storage.Config)
		if err != nil {
			return errors.New("failed to create storage: " + err.Error())
		}
		log.Info().EmbedObject(ps).Msg("started storage")
	}
	r.peerStore = ps

	preHooks, err := middleware.HooksFromHookConfigs(cfg.PreHooks)
	if err != nil {
		return errors.New("failed to validate hook config: " + err.Error())
	}
	postHooks, err := middleware.HooksFromHookConfigs(cfg.PostHooks)
	if err != nil {
		return errors.New("failed to validate hook config: " + err.Error())
	}

	log.Info().
		Strs("prehooks", cfg.PreHookNames()).
		Strs("posthooks", cfg.PostHookNames()).
		Msg("starting tracker logic")
	r.logic = middleware.NewLogic(cfg.ResponseConfig, r.peerStore, preHooks, postHooks)

	if cfg.HTTPConfig.Addr != "" {
		log.Info().EmbedObject(cfg.HTTPConfig).Msg("starting HTTP frontend")
		httpfe, err := http.NewFrontend(r.logic, cfg.HTTPConfig)
		if err != nil {
			return err
		}
		r.sg.Add(httpfe)
	}

	if cfg.UDPConfig.Addr != "" {
		log.Info().EmbedObject(cfg.UDPConfig).Msg("starting UDP frontend")
		udpfe, err := udp.NewFrontend(r.logic, cfg.UDPConfig)
		if err != nil {
			return err
		}
		r.sg.Add(udpfe)
	}

	return nil
}

func combineErrors(prefix string, errs []error) error {
	var errStrs []string
	for _, err := range errs {
		errStrs = append(errStrs, err.Error())
	}

	return errors.New(prefix + ": " + strings.Join(errStrs, "; "))
}

// Stop shuts down an instance of Chihaya.
func (r *Run) Stop(keepPeerStore bool) (storage.PeerStore, error) {
	log.Debug().Msg("stopping frontends and metrics server")
	if errs := r.sg.Stop().Wait(); len(errs) != 0 {
		return nil, combineErrors("failed while shutting down frontends", errs)
	}

	log.Debug().Msg("stopping logic")
	if errs := r.logic.Stop().Wait(); len(errs) != 0 {
		return nil, combineErrors("failed while shutting down middleware", errs)
	}

	if !keepPeerStore {
		log.Debug().Msg("stopping peer store")
		if errs := r.peerStore.Stop().Wait(); len(errs) != 0 {
			return nil, combineErrors("failed while shutting down peer store", errs)
		}
		r.peerStore = nil
	}

	return r.peerStore, nil
}

// RootRunCmdFunc implements a Cobra command that runs an instance of Chihaya
// and handles reloading and shutdown via process signals.
func RootRunCmdFunc(cmd *cobra.Command, args []string) error {
	configFilePath, err := cmd.Flags().GetString("config")
	if err != nil {
		return err
	}

	r, err := NewRun(configFilePath)
	if err != nil {
		return err
	}

	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	reload := makeReloadChan()

	for {
		select {
		case <-reload:
			log.Info().Msg("reloading; received SIGUSR1")
			peerStore, err := r.Stop(true)
			if err != nil {
				return err
			}

			if err := r.Start(peerStore); err != nil {
				return err
			}
		case <-quit:
			log.Info().Msg("shutting down; received SIGINT/SIGTERM")
			if _, err := r.Stop(false); err != nil {
				return err
			}

			return nil
		}
	}
}

func prettyLogPreRunE(cmd *cobra.Command, args []string) error {
	if isatty.IsTerminal(os.Stdout.Fd()) && !cobrautil.MustGetBool(cmd, "json") {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	}
	return nil
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "chihaya",
		Short: "BitTorrent Tracker",
		Long:  "A customizable, multi-protocol BitTorrent Tracker",
		RunE:  RootRunCmdFunc,
		PersistentPreRunE: cobrautil.CommandStack(
			cobrautil.SyncViperPreRunE("CHIHAYA"),
			prettyLogPreRunE,
			cobrautil.ZeroLogPreRunE,
		),
	}

	cobrautil.RegisterZeroLogFlags(rootCmd.PersistentFlags())
	rootCmd.PersistentFlags().Bool("json", false, "enable JSON logging")
	rootCmd.Flags().String("config", "/etc/chihaya.yaml", "location of configuration file")

	e2eCmd := &cobra.Command{
		Use:   "e2e",
		Short: "exec e2e tests",
		Long:  "Execute the Chihaya end-to-end test suite",
		RunE:  EndToEndRunCmdFunc,
	}

	e2eCmd.Flags().String("httpaddr", "http://127.0.0.1:6969/announce", "address of the HTTP tracker")
	e2eCmd.Flags().String("udpaddr", "udp://127.0.0.1:6969", "address of the UDP tracker")
	e2eCmd.Flags().Duration("delay", time.Second, "delay between announces")

	rootCmd.AddCommand(e2eCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal().Err(err).Msg("failed when executing root cobra command")
	}
}
