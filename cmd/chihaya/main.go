package main

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	httpfrontend "github.com/chihaya/chihaya/frontend/http"
	udpfrontend "github.com/chihaya/chihaya/frontend/udp"
	"github.com/chihaya/chihaya/middleware"
	"github.com/chihaya/chihaya/storage/memory"
)

type ConfigFile struct {
	MainConfigBlock struct {
		middleware.Config
		PrometheusAddr string              `yaml:"prometheus_addr"`
		HTTPConfig     httpfrontend.Config `yaml:"http"`
		UDPConfig      udpfrontend.Config  `yaml:"udp"`
		Storage        memory.Config       `yaml:"storage"`
	} `yaml:"chihaya"`
}

// ParseConfigFile returns a new ConfigFile given the path to a YAML
// configuration file.
//
// It supports relative and absolute paths and environment variables.
func ParseConfigFile(path string) (*ConfigFile, error) {
	if path == "" {
		return nil, errors.New("no config path specified")
	}

	f, err := os.Open(os.ExpandEnv(path))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	contents, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	var cfgFile ConfigFile
	err = yaml.Unmarshal(contents, &cfgFile)
	if err != nil {
		return nil, err
	}

	return &cfgFile, nil
}

func main() {
	var configFilePath string
	var cpuProfilePath string

	var rootCmd = &cobra.Command{
		Use:   "chihaya",
		Short: "BitTorrent Tracker",
		Long:  "A customizible, multi-protocol BitTorrent Tracker",
		Run: func(cmd *cobra.Command, args []string) {
			if err := func() error {
				if cpuProfilePath != "" {
					log.Println("enabled CPU profiling to " + cpuProfilePath)
					f, err := os.Create(cpuProfilePath)
					if err != nil {
						return err
					}
					pprof.StartCPUProfile(f)
					defer pprof.StopCPUProfile()
				}

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
					log.Println("started serving prometheus stats on", cfg.PrometheusAddr)
					if err := promServer.ListenAndServe(); err != nil {
						log.Fatal(err)
					}
				}()

				// Force the compiler to enforce memory against the storage interface.
				peerStore, err := memory.New(cfg.Storage)
				if err != nil {
					return err
				}

				// TODO create Hooks
				logic := middleware.NewLogic(cfg.Config, peerStore, nil, nil, nil, nil)
				if err != nil {
					return err
				}

				shutdown := make(chan struct{})
				errChan := make(chan error)

				var httpFrontend *httpfrontend.Frontend
				var udpFrontend *udpfrontend.Frontend

				if cfg.HTTPConfig.Addr != "" {
					httpFrontend = httpfrontend.NewFrontend(logic, cfg.HTTPConfig)

					go func() {
						log.Println("started serving HTTP on", cfg.HTTPConfig.Addr)
						if err := httpFrontend.ListenAndServe(); err != nil {
							errChan <- errors.New("failed to cleanly shutdown HTTP frontend: " + err.Error())
						}
					}()
				}

				if cfg.UDPConfig.Addr != "" {
					udpFrontend = udpfrontend.NewFrontend(logic, cfg.UDPConfig)

					go func() {
						log.Println("started serving UDP on", cfg.UDPConfig.Addr)
						if err := udpFrontend.ListenAndServe(); err != nil {
							errChan <- errors.New("failed to cleanly shutdown UDP frontend: " + err.Error())
						}
					}()
				}

				sigChan := make(chan os.Signal)
				signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
				go func() {
					select {
					case <-sigChan:
					case <-shutdown:
					}

					if udpFrontend != nil {
						udpFrontend.Stop()
					}

					if httpFrontend != nil {
						httpFrontend.Stop()
					}

					for err := range peerStore.Stop() {
						if err != nil {
							errChan <- err
						}
					}

					close(errChan)
				}()

				closed := false
				var bufErr error
				for err = range errChan {
					if err != nil {
						if !closed {
							close(shutdown)
							closed = true
						} else {
							log.Println(bufErr)
						}
						bufErr = err
					}
				}

				return bufErr
			}(); err != nil {
				log.Fatal(err)
			}
		},
	}

	rootCmd.Flags().StringVar(&configFilePath, "config", "/etc/chihaya.yaml", "location of configuration file")
	rootCmd.Flags().StringVarP(&cpuProfilePath, "cpuprofile", "", "", "location to save a CPU profile")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
