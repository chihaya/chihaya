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

	"github.com/jzelinskie/trakr"
)

type ConfigFile struct {
	Config struct {
		PrometheusAddr string `yaml:"prometheus_addr"`
		trakr.MultiTracker
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

				go func() {
					shutdown := make(chan os.Signal)
					signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
					<-shutdown
					configFile.Config.MultiTracker.Stop()
				}()

				if err := configFile.Config.MultiTracker.ListenAndServe(); err != nil {
					return errors.New("failed to cleanly shutdown: " + err.Error())
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
