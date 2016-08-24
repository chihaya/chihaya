package main

import (
	"errors"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"

	httpfrontend "github.com/chihaya/chihaya/frontend/http"
	udpfrontend "github.com/chihaya/chihaya/frontend/udp"
	"github.com/chihaya/chihaya/middleware"
	"github.com/chihaya/chihaya/storage/memory"
)

// ConfigFile represents a namespaced YAML configation file.
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
