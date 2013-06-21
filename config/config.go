package config

import (
	"encoding/json"
	"os"
	"time"
)

type Config struct {
	Addr      string  `json:"addr"`
	Storage   Storage `json:"storage"`
	Private   bool    `json:"private"`
	Freeleech bool    `json:"freeleech"`

	Announce       Duration `json:"announce"`
	MinAnnounce    Duration `json:"min_announce"`
	BufferPoolSize int      `json:"bufferpool_size"`

	Whitelist []string `json:"whitelist"`
}

// StorageConfig represents the settings used for storage or cache.
type Storage struct {
	Driver   string `json:"driver"`
	Protocol string `json:"protocol"`
	Addr     string `json:"addr"`
	Username string `json:"user"`
	Password string `json:"pass"`
	Schema   string `json:"schema"`
	Encoding string `json:"encoding"`
}

type Duration struct {
	time.Duration
}

func (d *Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var str string
	err := json.Unmarshal(b, &str)
	d.Duration, err = time.ParseDuration(str)
	return err
}

func New(path string) (*Config, error) {
	expandedPath := os.ExpandEnv(path)
	f, err := os.Open(expandedPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	conf := &Config{}
	err = json.NewDecoder(f).Decode(conf)
	if err != nil {
		return nil, err
	}
	return conf, nil
}
