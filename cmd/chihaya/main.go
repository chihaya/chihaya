package main

import (
	"errors"
	"net/http"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"

	httpfrontend "github.com/chihaya/chihaya/frontend/http"
	udpfrontend "github.com/chihaya/chihaya/frontend/udp"
	"github.com/chihaya/chihaya/middleware"
	"github.com/chihaya/chihaya/storage"
	"github.com/chihaya/chihaya/storage/memory"
)

func rootCmdRun(cmd *cobra.Command, args []string) error {
	debugLog, _ := cmd.Flags().GetBool("debug")
	if debugLog {
		log.SetLevel(log.DebugLevel)
		log.Debugln("debug logging enabled")
	}
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

	configFilePath, _ := cmd.Flags().GetString("config")
	configFile, err := ParseConfigFile(configFilePath)
	if err != nil {
		return errors.New("failed to read config: " + err.Error())
	}
	cfg := configFile.MainConfigBlock

	go func() {
		promServer := http.Server{
			Addr:    cfg.PrometheusAddr,
			Handler: prometheus.Handler(),
		}
		log.Infoln("started serving prometheus stats on", cfg.PrometheusAddr)
		if err := promServer.ListenAndServe(); err != nil {
			log.Fatalln("failed to start prometheus server:", err.Error())
		}
	}()

	peerStore, err := memory.New(cfg.Storage)
	if err != nil {
		return errors.New("failed to create memory storage: " + err.Error())
	}

	preHooks, postHooks, err := configFile.CreateHooks()
	if err != nil {
		return errors.New("failed to create hooks: " + err.Error())
	}

	logic := middleware.NewLogic(cfg.Config, peerStore, preHooks, postHooks)

	errChan := make(chan error)

	httpFrontend, udpFrontend := startFrontends(cfg.HTTPConfig, cfg.UDPConfig, logic, errChan)

	shutdown := make(chan struct{})
	quit := make(chan os.Signal)
	restart := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	signal.Notify(restart, syscall.SIGUSR1)

	go func() {
		for {
			select {
			case <-restart:
				log.Info("Got signal to restart")

				// Reload config
				configFile, err = ParseConfigFile(configFilePath)
				if err != nil {
					log.Error("failed to read config: " + err.Error())
				}
				cfg = configFile.MainConfigBlock

				preHooks, postHooks, err = configFile.CreateHooks()
				if err != nil {
					log.Error("failed to create hooks: " + err.Error())
				}

				// Stop frontends and logic
				stopFrontends(udpFrontend, httpFrontend)

				stopLogic(logic, errChan)

				// Restart
				log.Debug("Restarting logic")
				logic = middleware.NewLogic(cfg.Config, peerStore, preHooks, postHooks)

				log.Debug("Restarting frontends")
				httpFrontend, udpFrontend = startFrontends(cfg.HTTPConfig, cfg.UDPConfig, logic, errChan)

				log.Debug("Successfully restarted")

			case <-quit:
				stop(udpFrontend, httpFrontend, logic, errChan, peerStore)
			case <-shutdown:
				stop(udpFrontend, httpFrontend, logic, errChan, peerStore)
			}
		}
	}()

	closed := false
	var bufErr error
	for err = range errChan {
		if err != nil {
			if !closed {
				close(shutdown)
				closed = true
			} else {
				log.Infoln(bufErr)
			}
			bufErr = err
		}
	}

	return bufErr
}

func stopFrontends(udpFrontend *udpfrontend.Frontend, httpFrontend *httpfrontend.Frontend) {
	log.Debug("Stopping frontends")
	if udpFrontend != nil {
		udpFrontend.Stop()
	}

	if httpFrontend != nil {
		httpFrontend.Stop()
	}
}

func stopLogic(logic *middleware.Logic, errChan chan error) {
	log.Debug("Stopping logic")
	errs := logic.Stop()
	for _, err := range errs {
		errChan <- err
	}
}

func stop(udpFrontend *udpfrontend.Frontend, httpFrontend *httpfrontend.Frontend, logic *middleware.Logic, errChan chan error, peerStore storage.PeerStore) {
	stopFrontends(udpFrontend, httpFrontend)

	stopLogic(logic, errChan)

	// Stop storage
	log.Debug("Stopping storage")
	for err := range peerStore.Stop() {
		if err != nil {
			errChan <- err
		}
	}

	close(errChan)
}

func startFrontends(httpConfig httpfrontend.Config, udpConfig udpfrontend.Config, logic *middleware.Logic, errChan chan<- error) (httpFrontend *httpfrontend.Frontend, udpFrontend *udpfrontend.Frontend) {
	if httpConfig.Addr != "" {
		httpFrontend = httpfrontend.NewFrontend(logic, httpConfig)

		go func() {
			log.Infoln("started serving HTTP on", httpConfig.Addr)
			if err := httpFrontend.ListenAndServe(); err != nil {
				errChan <- errors.New("failed to cleanly shutdown HTTP frontend: " + err.Error())
			}
		}()
	}

	if udpConfig.Addr != "" {
		udpFrontend = udpfrontend.NewFrontend(logic, udpConfig)

		go func() {
			log.Infoln("started serving UDP on", udpConfig.Addr)
			if err := udpFrontend.ListenAndServe(); err != nil {
				errChan <- errors.New("failed to cleanly shutdown UDP frontend: " + err.Error())
			}
		}()
	}

	return
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "chihaya",
		Short: "BitTorrent Tracker",
		Long:  "A customizable, multi-protocol BitTorrent Tracker",
		Run: func(cmd *cobra.Command, args []string) {
			if err := rootCmdRun(cmd, args); err != nil {
				log.Fatal(err)
			}
		},
	}
	rootCmd.Flags().String("config", "/etc/chihaya.yaml", "location of configuration file")
	rootCmd.Flags().String("cpuprofile", "", "location to save a CPU profile")
	rootCmd.Flags().Bool("debug", false, "enable debug logging")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
