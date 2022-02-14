package main

import (
	"errors"
	"io/ioutil"
	"os"

	yaml "gopkg.in/yaml.v2"

	"github.com/chihaya/chihaya/frontend/http"
	"github.com/chihaya/chihaya/frontend/udp"
	"github.com/chihaya/chihaya/middleware"

	// Imports to register middleware drivers.
	_ "github.com/chihaya/chihaya/middleware/clientapproval"
	_ "github.com/chihaya/chihaya/middleware/jwt"
	_ "github.com/chihaya/chihaya/middleware/torrentapproval"
	_ "github.com/chihaya/chihaya/middleware/varinterval"

	// Imports to register storage drivers.
	_ "github.com/chihaya/chihaya/storage/database"
	_ "github.com/chihaya/chihaya/storage/memory"
	_ "github.com/chihaya/chihaya/storage/redis"
)

type storageConfig struct {
	Name   string      `yaml:"name"`
	Config interface{} `yaml:"config"`
}

// Config represents the configuration used for executing Chihaya.
type Config struct {
	middleware.ResponseConfig `yaml:",inline"`
	MetricsAddr               string                  `yaml:"metrics_addr"`
	HTTPConfig                http.Config             `yaml:"http"`
	UDPConfig                 udp.Config              `yaml:"udp"`
	Storage                   storageConfig           `yaml:"storage"`
	PreHooks                  []middleware.HookConfig `yaml:"prehooks"`
	PostHooks                 []middleware.HookConfig `yaml:"posthooks"`
}

// PreHookNames returns only the names of the configured middleware.
func (cfg Config) PreHookNames() (names []string) {
	for _, hook := range cfg.PreHooks {
		names = append(names, hook.Name)
	}

	return
}

// PostHookNames returns only the names of the configured middleware.
func (cfg Config) PostHookNames() (names []string) {
	for _, hook := range cfg.PostHooks {
		names = append(names, hook.Name)
	}

	return
}

// ConfigFile represents a namespaced YAML configation file.
type ConfigFile struct {
	Chihaya Config `yaml:"chihaya"`
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
