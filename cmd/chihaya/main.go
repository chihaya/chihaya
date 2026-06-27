// Package main provides argument parsing and execution logic for the command line tool.
package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/chihaya/chihaya/frontend/http"
	"github.com/chihaya/chihaya/frontend/udp"
	"github.com/chihaya/chihaya/middleware"
	"github.com/chihaya/chihaya/pkg/metrics"
	"github.com/chihaya/chihaya/pkg/slog"
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

	slog.Info("starting metrics server", slog.String("addr", cfg.MetricsAddr))
	r.sg.Add(metrics.NewServer(cfg.MetricsAddr))

	if ps == nil {
		slog.Info("starting storage", slog.String("name", cfg.Storage.Name))
		ps, err = storage.NewPeerStore(cfg.Storage.Name, cfg.Storage.Config)
		if err != nil {
			return errors.New("failed to create storage: " + err.Error())
		}
		slog.Info("started storage", slog.Valuer("peerStore", ps))
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

	slog.Info(
		"starting tracker logic",
		slog.Any("prehooks", cfg.PreHookNames()),
		slog.Any("posthooks", cfg.PostHookNames()),
	)
	r.logic = middleware.NewLogic(cfg.ResponseConfig, r.peerStore, preHooks, postHooks)

	if cfg.HTTPConfig.Addr != "" {
		slog.Info("starting HTTP frontend", slog.Valuer("config", &cfg.HTTPConfig))
		httpfe, err := http.NewFrontend(r.logic, cfg.HTTPConfig)
		if err != nil {
			return err
		}
		r.sg.Add(httpfe)
	}

	if cfg.UDPConfig.Addr != "" {
		slog.Info("starting UDP frontend", slog.Valuer("config", &cfg.UDPConfig))
		udpfe, err := udp.NewFrontend(r.logic, cfg.UDPConfig)
		if err != nil {
			return err
		}
		r.sg.Add(udpfe)
	}

	return nil
}

func combineErrors(prefix string, errs []error) error {
	errStrs := make([]string, 0, len(errs))
	for _, err := range errs {
		errStrs = append(errStrs, err.Error())
	}

	return errors.New(prefix + ": " + strings.Join(errStrs, "; "))
}

// Stop shuts down an instance of Chihaya.
func (r *Run) Stop(keepPeerStore bool) (storage.PeerStore, error) {
	slog.Debug("stopping frontends and metrics server")
	if errs := r.sg.Stop().Wait(); len(errs) != 0 {
		return nil, combineErrors("failed while shutting down frontends", errs)
	}

	slog.Debug("stopping logic")
	if errs := r.logic.Stop().Wait(); len(errs) != 0 {
		return nil, combineErrors("failed while shutting down middleware", errs)
	}

	if !keepPeerStore {
		slog.Debug("stopping peer store")
		if errs := r.peerStore.Stop().Wait(); len(errs) != 0 {
			return nil, combineErrors("failed while shutting down peer store", errs)
		}
		r.peerStore = nil
	}

	return r.peerStore, nil
}

// RootRunCmdFunc implements a Cobra command that runs an instance of Chihaya
// and handles reloading and shutdown via process signals.
func RootRunCmdFunc(cmd *cobra.Command, _ []string) error {
	configFilePath, err := cmd.Flags().GetString("config")
	if err != nil {
		return err
	}

	r, err := NewRun(configFilePath)
	if err != nil {
		return err
	}

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	reload, _ := signal.NotifyContext(context.Background(), ReloadSignals...)

	for {
		select {
		case <-reload.Done():
			slog.Info("reloading; received reload signal")
			peerStore, err := r.Stop(true)
			if err != nil {
				return err
			}

			if err := r.Start(peerStore); err != nil {
				return err
			}
		case <-ctx.Done():
			slog.Info("shutting down; received shutdown signal")
			if _, err := r.Stop(false); err != nil {
				return err
			}

			return nil
		}
	}
}

// RootPreRunCmdFunc handles command line flags for the Run command.
func RootPreRunCmdFunc(cmd *cobra.Command, _ []string) error {
	defer slog.Debug("logging configured")
	level := slog.LevelInfo
	if debugLog, err := cmd.Flags().GetBool("debug"); err != nil {
		return err
	} else if debugLog {
		level = slog.LevelDebug
	}

	if jsonLog, err := cmd.Flags().GetBool("json"); err != nil {
		return err
	} else if jsonLog {
		slog.SetDefaultHandler(slog.NewJSONHandler(os.Stderr, level))
		return nil
	}

	noColor := !isatty.IsTerminal(os.Stderr.Fd())
	if noColorsFlag, err := cmd.Flags().GetBool("nocolors"); err != nil {
		return err
	} else if noColorsFlag {
		noColor = true
	}

	slog.SetDefaultHandler(slog.NewTextHandler(os.Stderr, level, noColor))
	return nil
}

// RootPostRunCmdFunc handles clean up of any state initialized by command line
// flags.
func RootPostRunCmdFunc(_ *cobra.Command, _ []string) error { return nil }

func main() {
	rootCmd := &cobra.Command{
		Use:                "chihaya",
		Short:              "BitTorrent Tracker",
		Long:               "A customizable, multi-protocol BitTorrent Tracker",
		PersistentPreRunE:  RootPreRunCmdFunc,
		RunE:               RootRunCmdFunc,
		PersistentPostRunE: RootPostRunCmdFunc,
	}

	rootCmd.PersistentFlags().Bool("debug", false, "enable debug logging")
	rootCmd.PersistentFlags().Bool("json", false, "enable json logging")
	if runtime.GOOS == "windows" {
		rootCmd.PersistentFlags().Bool("nocolors", true, "disable log coloring")
	} else {
		rootCmd.PersistentFlags().Bool("nocolors", false, "disable log coloring")
	}

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
		slog.Error("failed when executing root cobra command", slog.Err(err))
		os.Exit(1)
	}
}
