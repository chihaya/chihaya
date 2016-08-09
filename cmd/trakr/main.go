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

	"github.com/jzelinskie/trakr/backend"
	httpfrontend "github.com/jzelinskie/trakr/frontend/http"
	udpfrontend "github.com/jzelinskie/trakr/frontend/udp"
)

type ConfigFile struct {
	Config struct {
		PrometheusAddr string `yaml:"prometheus_addr"`
		backend.BackendConfig
		HTTPConfig httpfrontend.Config `yaml:"http"`
		UDPConfig  udpfrontend.Config  `yaml:"udp"`
	} `yaml:"trakr"`
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
		Use:   "trakr",
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

				go func() {
					promServer := http.Server{
						Addr:    configFile.Config.PrometheusAddr,
						Handler: prometheus.Handler(),
					}
					log.Println("started serving prometheus stats on", configFile.Config.PrometheusAddr)
					if err := promServer.ListenAndServe(); err != nil {
						log.Fatal(err)
					}
				}()

				// TODO create PeerStore
				// TODO create Hooks
				trackerBackend, err := backend.New(configFile.Config.BackendConfig, nil, nil, nil, nil, nil)
				if err != nil {
					return err
				}

				errChan := make(chan error)
				closedChan := make(chan struct{})

				var hFrontend *httpfrontend.Frontend
				var uFrontend *udpfrontend.Frontend

				if configFile.Config.HTTPConfig.Addr != "" {
					// TODO get the real TrackerFuncs
					hFrontend = httpfrontend.NewFrontend(trackerBackend, configFile.Config.HTTPConfig)

					go func() {
						log.Println("started serving HTTP on", configFile.Config.HTTPConfig.Addr)
						if err := hFrontend.ListenAndServe(); err != nil {
							errChan <- errors.New("failed to cleanly shutdown HTTP frontend: " + err.Error())
						}
					}()
				}

				if configFile.Config.UDPConfig.Addr != "" {
					// TODO get the real TrackerFuncs
					uFrontend = udpfrontend.NewFrontend(trackerBackend, configFile.Config.UDPConfig)

					go func() {
						log.Println("started serving UDP on", configFile.Config.UDPConfig.Addr)
						if err := uFrontend.ListenAndServe(); err != nil {
							errChan <- errors.New("failed to cleanly shutdown UDP frontend: " + err.Error())
						}
					}()
				}

				shutdown := make(chan os.Signal)
				signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
				go func() {
					<-shutdown

					if uFrontend != nil {
						uFrontend.Stop()
					}

					if hFrontend != nil {
						hFrontend.Stop()
					}

					// TODO: stop PeerStore

					close(errChan)
					close(closedChan)
				}()

				for err := range errChan {
					if err != nil {
						close(shutdown)
						<-closedChan
						return err
					}
				}

				return nil
			}(); err != nil {
				log.Fatal(err)
			}
		},
	}

	rootCmd.Flags().StringVar(&configFilePath, "config", "/etc/trakr.yaml", "location of configuration file (defaults to /etc/trakr.yaml)")
	rootCmd.Flags().StringVarP(&cpuProfilePath, "cpuprofile", "", "", "location to save a CPU profile")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
