package main

import (
	"errors"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"

	httpfrontend "github.com/chihaya/chihaya/frontend/http"
	udpfrontend "github.com/chihaya/chihaya/frontend/udp"
	"github.com/chihaya/chihaya/middleware"
	"github.com/chihaya/chihaya/middleware/clientapproval"
	"github.com/chihaya/chihaya/middleware/jwt"
	"github.com/chihaya/chihaya/middleware/varinterval"
	"github.com/chihaya/chihaya/storage/memory"
)

type hookConfig struct {
	Name   string      `yaml:"name"`
	Config interface{} `yaml:"config"`
}

type hookConfigs []hookConfig

// Names returns all hook names listed in the configuration.
func (hookCfgs hookConfigs) Names() (hookNames []string) {
	hookNames = make([]string, len(hookCfgs))
	for index, hookCfg := range hookCfgs {
		hookNames[index] = hookCfg.Name
	}

	return
}

// Config represents the configuration used for executing Chihaya.
type Config struct {
	middleware.Config `yaml:",inline"`
	PrometheusAddr    string              `yaml:"prometheus_addr"`
	HTTPConfig        httpfrontend.Config `yaml:"http"`
	UDPConfig         udpfrontend.Config  `yaml:"udp"`
	Storage           memory.Config       `yaml:"storage"`
	PreHooks          hookConfigs         `yaml:"prehooks"`
	PostHooks         hookConfigs         `yaml:"posthooks"`
}

// CreateHooks creates instances of Hooks for all of the PreHooks and PostHooks
// configured in a Config.
func (cfg Config) CreateHooks() (preHooks, postHooks []middleware.Hook, err error) {
	for _, hookCfg := range cfg.PreHooks {
		cfgBytes, err := yaml.Marshal(hookCfg.Config)
		if err != nil {
			panic("failed to remarshal valid YAML")
		}

		switch hookCfg.Name {
		case "jwt":
			var jwtCfg jwt.Config
			err := yaml.Unmarshal(cfgBytes, &jwtCfg)
			if err != nil {
				return nil, nil, errors.New("invalid JWT middleware config: " + err.Error())
			}
			hook, err := jwt.NewHook(jwtCfg)
			if err != nil {
				return nil, nil, errors.New("invalid JWT middleware config: " + err.Error())
			}
			preHooks = append(preHooks, hook)
		case "client approval":
			var caCfg clientapproval.Config
			err := yaml.Unmarshal(cfgBytes, &caCfg)
			if err != nil {
				return nil, nil, errors.New("invalid client approval middleware config: " + err.Error())
			}
			hook, err := clientapproval.NewHook(caCfg)
			if err != nil {
				return nil, nil, errors.New("invalid client approval middleware config: " + err.Error())
			}
			preHooks = append(preHooks, hook)
		case "interval variation":
			var viCfg varinterval.Config
			err := yaml.Unmarshal(cfgBytes, &viCfg)
			if err != nil {
				return nil, nil, errors.New("invalid interval variation middleware config: " + err.Error())
			}
			hook, err := varinterval.New(viCfg)
			if err != nil {
				return nil, nil, errors.New("invalid interval variation middleware config: " + err.Error())
			}
			preHooks = append(preHooks, hook)
		}
	}

	for _, hookCfg := range cfg.PostHooks {
		switch hookCfg.Name {
		}
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
