package main

import (
	"errors"
	"os"
	"os/signal"
	"runtime/pprof"
	"strings"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/chihaya/chihaya/frontend/http"
	"github.com/chihaya/chihaya/frontend/udp"
	"github.com/chihaya/chihaya/middleware"
	"github.com/chihaya/chihaya/pkg/log"
	"github.com/chihaya/chihaya/pkg/prometheus"
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

	log.Info("starting Prometheus server", log.Fields{"addr": cfg.PrometheusAddr})
	r.sg.Add(prometheus.NewServer(cfg.PrometheusAddr))

	if ps == nil {
		log.Info("starting storage", log.Fields{"name": cfg.Storage.Name})
		ps, err = storage.NewPeerStore(cfg.Storage.Name, cfg.Storage.Config)
		if err != nil {
			return errors.New("failed to create storage: " + err.Error())
		}
		log.Info("started storage", ps.LogFields())
	}
	r.peerStore = ps

	preHooks, postHooks, err := cfg.CreateHooks()
	if err != nil {
		return errors.New("failed to validate hook config: " + err.Error())
	}
	log.Info("starting middleware", log.Fields{
		"preHooks":  cfg.PreHooks.Names(),
		"postHooks": cfg.PostHooks.Names(),
	})
	r.logic = middleware.NewLogic(cfg.Config, r.peerStore, preHooks, postHooks)

	if cfg.HTTPConfig.Addr != "" {
		log.Info("starting HTTP frontend", cfg.HTTPConfig.LogFields())
		httpfe, err := http.NewFrontend(r.logic, &cfg.RequestSanitizer, cfg.HTTPConfig)
		if err != nil {
			return err
		}
		r.sg.Add(httpfe)
	}

	if cfg.UDPConfig.Addr != "" {
		log.Info("starting UDP frontend", cfg.UDPConfig.LogFields())
		udpfe, err := udp.NewFrontend(r.logic, &cfg.RequestSanitizer, cfg.UDPConfig)
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
	log.Debug("stopping frontends and prometheus endpoint")
	if errs := r.sg.Stop(); len(errs) != 0 {
		return nil, combineErrors("failed while shutting down frontends", errs)
	}

	log.Debug("stopping logic")
	if errs := r.logic.Stop(); len(errs) != 0 {
		return nil, combineErrors("failed while shutting down middleware", errs)
	}

	if !keepPeerStore {
		log.Debug("stopping peer store")
		if err, closed := <-r.peerStore.Stop(); !closed {
			return nil, err
		}
		r.peerStore = nil
	}

	return r.peerStore, nil
}

// RunCmdFunc implements a Cobra command that runs an instance of Chihaya and
// handles reloading and shutdown via process signals.
func RunCmdFunc(cmd *cobra.Command, args []string) error {
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

	reload := make(chan os.Signal)
	signal.Notify(reload, syscall.SIGUSR1)

	for {
		select {
		case <-reload:
			log.Info("reloading; received SIGUSR1")
			peerStore, err := r.Stop(true)
			if err != nil {
				return err
			}

			if err := r.Start(peerStore); err != nil {
				return err
			}
		case <-quit:
			log.Info("shutting down; received SIGINT/SIGTERM")
			if _, err := r.Stop(false); err != nil {
				return err
			}

			return nil
		}
	}
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "chihaya",
		Short: "BitTorrent Tracker",
		Long:  "A customizable, multi-protocol BitTorrent Tracker",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			jsonLog, err := cmd.Flags().GetBool("json")
			if err != nil {
				return err
			}
			if jsonLog {
				log.SetFormatter(&logrus.JSONFormatter{})
			}

			debugLog, err := cmd.Flags().GetBool("debug")
			if err != nil {
				return err
			}
			if debugLog {
				log.Info("enabling debug logging")
				log.SetDebug(true)
			}

			cpuProfilePath, err := cmd.Flags().GetString("cpuprofile")
			if err != nil {
				return err
			}
			if cpuProfilePath != "" {
				log.Info("enabling CPU profiling", log.Fields{"path": cpuProfilePath})
				f, err := os.Create(cpuProfilePath)
				if err != nil {
					return err
				}
				pprof.StartCPUProfile(f)
			}

			return nil
		},
		RunE: RunCmdFunc,
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			// StopCPUProfile() noops when not profiling.
			pprof.StopCPUProfile()
			return nil
		},
	}
	rootCmd.Flags().String("config", "/etc/chihaya.yaml", "location of configuration file")
	rootCmd.Flags().String("cpuprofile", "", "location to save a CPU profile")
	rootCmd.Flags().Bool("debug", false, "enable debug logging")
	rootCmd.Flags().Bool("json", false, "enable json logging")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal("failed when executing root cobra command: " + err.Error())
	}
}
