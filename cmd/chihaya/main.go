package main

import (
	"errors"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/chihaya/chihaya/frontend/http"
	"github.com/chihaya/chihaya/frontend/udp"
	"github.com/chihaya/chihaya/middleware"
	"github.com/chihaya/chihaya/pkg/prometheus"
	"github.com/chihaya/chihaya/pkg/stop"
	"github.com/chihaya/chihaya/storage"
	"github.com/chihaya/chihaya/storage/memory"
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

	return r, r.Start()
}

// Start begins an instance of Chihaya.
func (r *Run) Start() error {
	configFile, err := ParseConfigFile(r.configFilePath)
	if err != nil {
		return errors.New("failed to read config: " + err.Error())
	}

	chihayaCfg := configFile.Chihaya
	preHooks, postHooks, err := chihayaCfg.CreateHooks()
	if err != nil {
		return errors.New("failed to validate hook config: " + err.Error())
	}

	r.sg = stop.NewGroup()
	r.sg.Add(prometheus.NewServer(chihayaCfg.PrometheusAddr))

	r.peerStore, err = memory.New(chihayaCfg.Storage)
	if err != nil {
		return errors.New("failed to create memory storage: " + err.Error())
	}

	r.logic = middleware.NewLogic(chihayaCfg.Config, r.peerStore, preHooks, postHooks)

	if chihayaCfg.HTTPConfig.Addr != "" {
		httpfe, err := http.NewFrontend(r.logic, chihayaCfg.HTTPConfig)
		if err != nil {
			return err
		}
		r.sg.Add(httpfe)
	}

	if chihayaCfg.UDPConfig.Addr != "" {
		udpfe, err := udp.NewFrontend(r.logic, chihayaCfg.UDPConfig)
		if err != nil {
			return err
		}
		r.sg.Add(udpfe)
	}

	return nil
}

// Stop shuts down an instance of Chihaya.
func (r *Run) Stop() error {
	log.Debug("stopping frontends and prometheus endpoint")
	if errs := r.sg.Stop(); len(errs) != 0 {
		errDelimiter := "; "
		errStr := "failed while shutting down frontends: "

		for _, err := range errs {
			errStr += err.Error() + errDelimiter
		}

		// Remove the last delimiter.
		errStr = errStr[0 : len(errStr)-len(errDelimiter)]

		return errors.New(errStr)
	}

	log.Debug("stopping logic")
	if errs := r.logic.Stop(); len(errs) != 0 {
		errDelimiter := "; "
		errStr := "failed while shutting down middleware: "

		for _, err := range errs {
			errStr += err.Error() + errDelimiter
		}

		// Remove the last delimiter.
		errStr = errStr[0 : len(errStr)-len(errDelimiter)]

		return errors.New(errStr)
	}

	log.Debug("stopping peer store")
	if err, closed := <-r.peerStore.Stop(); !closed {
		return err
	}

	return nil
}

// RunCmdFunc implements a Cobra command that runs an instance of Chihaya and
// handles reloading and shutdown via process signals.
func RunCmdFunc(cmd *cobra.Command, args []string) error {
	cpuProfilePath, _ := cmd.Flags().GetString("cpuprofile")
	if cpuProfilePath != "" {
		log.Infoln("enabled CPU profiling to", cpuProfilePath)
		f, err := os.Create(cpuProfilePath)
		if err != nil {
			return err
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

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
			log.Info("received SIGUSR1")
			if err := r.Stop(); err != nil {
				return err
			}
			if err := r.Start(); err != nil {
				return err
			}
		case <-quit:
			log.Info("received SIGINT/SIGTERM")
			if err := r.Stop(); err != nil {
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
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			debugLog, _ := cmd.Flags().GetBool("debug")
			if debugLog {
				log.SetLevel(log.DebugLevel)
				log.Debugln("debug logging enabled")
			}
		},
		RunE: RunCmdFunc,
	}
	rootCmd.Flags().String("config", "/etc/chihaya.yaml", "location of configuration file")
	rootCmd.Flags().String("cpuprofile", "", "location to save a CPU profile")
	rootCmd.Flags().Bool("debug", false, "enable debug logging")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal("failed when executing root cobra command: " + err.Error())
	}
}
