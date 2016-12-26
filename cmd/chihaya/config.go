package main

import (
	"errors"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"

	"github.com/RealImage/chihaya/storage/redis"
	httpfrontend "github.com/chihaya/chihaya/frontend/http"
	udpfrontend "github.com/chihaya/chihaya/frontend/udp"
	"github.com/chihaya/chihaya/middleware"
	"github.com/chihaya/chihaya/middleware/clientapproval"
	"github.com/chihaya/chihaya/middleware/jwt"
	"github.com/chihaya/chihaya/storage"
	"github.com/chihaya/chihaya/storage/memory"
)

type hookConfig struct {
	Name   string      `yaml:"name"`
	Config interface{} `yaml:"config"`
}

type Storage struct {
	Type   string        `yaml:"type"`
	Config yaml.MapSlice `yaml:"config"`
}

// ConfigFile represents a namespaced YAML configation file.
type ConfigFile struct {
	MainConfigBlock struct {
		middleware.Config `yaml:",inline"`
		PrometheusAddr    string              `yaml:"prometheus_addr"`
		HTTPConfig        httpfrontend.Config `yaml:"http"`
		UDPConfig         udpfrontend.Config  `yaml:"udp"`
		Storage           Storage             `yaml:"storage"`
		PreHooks          []hookConfig        `yaml:"prehooks"`
		PostHooks         []hookConfig        `yaml:"posthooks"`
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

func (cfg ConfigFile) CreateStorage() (storage.PeerStore, error) {
	storage, err := yaml.Marshal(&cfg.MainConfigBlock.Storage.Config)
	if err != nil {
		return nil, err
	}
	switch cfg.MainConfigBlock.Storage.Type {
	case "memory":
		var mem memory.Config
		err := yaml.Unmarshal(storage, &mem)
		if err != nil {
			return nil, err
		}
		peerStore, err := memory.New(mem)
		if err != nil {
			return nil, err
		}
		return peerStore, nil
	case "redis":
		var red redis.Config
		err := yaml.Unmarshal(storage, &red)
		if err != nil {
			return nil, err
		}
		peerStore, err := redis.New(red)
		if err != nil {
			return nil, err
		}
		return peerStore, nil
	}
	return nil, err
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
		}
	}

	for _, hookCfg := range cfg.MainConfigBlock.PostHooks {
		switch hookCfg.Name {
		}
	}

	return
}
