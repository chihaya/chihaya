package main

import (
	"errors"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"

	httpfrontend "github.com/chihaya/chihaya/frontend/http"
	udpfrontend "github.com/chihaya/chihaya/frontend/udp"
	"github.com/chihaya/chihaya/middleware"
	"github.com/chihaya/chihaya/middleware/jwt"
	"github.com/chihaya/chihaya/storage/memory"
)

type hookConfig struct {
	Name   string      `yaml:"name"`
	Config interface{} `yaml:"config"`
}

// ConfigFile represents a namespaced YAML configation file.
type ConfigFile struct {
	MainConfigBlock struct {
		middleware.Config
		PrometheusAddr string              `yaml:"prometheus_addr"`
		HTTPConfig     httpfrontend.Config `yaml:"http"`
		UDPConfig      udpfrontend.Config  `yaml:"udp"`
		Storage        memory.Config       `yaml:"storage"`
		PreHooks       []hookConfig        `yaml:"prehooks"`
		PostHooks      []hookConfig        `yaml:"posthooks"`
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

// CreateHooks creates instances of Hooks for all of the PreHooks and PostHooks
// configured in a ConfigFile.
func (cfg ConfigFile) CreateHooks() (preHooks, postHooks []middleware.Hook, err error) {
	for _, hookCfg := range cfg.MainConfigBlock.PreHooks {
		cfgBytes, err := yaml.Marshal(hookCfg.Config)
		if err != nil {
			panic("failed to remarshal valid YAML")
		}

		switch hookCfg.Name {
		case "jwt":
			var jwtCfg jwt.Config
			err := yaml.Unmarshal(cfgBytes, &jwtCfg)
			if err != nil {
				return nil, nil, errors.New("invalid JWT middleware config" + err.Error())
			}
			preHooks = append(preHooks, jwt.NewHook(jwtCfg))
		}
	}

	for _, hookCfg := range cfg.MainConfigBlock.PostHooks {
		switch hookCfg.Name {
		}
	}

	return
}
