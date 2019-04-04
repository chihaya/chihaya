package config

import (
	"github.com/kelseyhightower/envconfig"
)

// DatabaseCfg ...
type DatabaseCfg struct {
	Host     string `envconfig:"MONGO_HOST"`
	Database string `envconfig:"MONGO_DB"`
	User     string `envconfig:"MONGO_USER"`
	Password string `envconfig:"MONGO_PASSWORD"`
}

// ServiceCfg ...
type ServiceCfg struct {
	HttpScheme      string `envconfig:"HTTP_SCHEME"`
	ServicePort     int    `envconfig:"SERVICE_PORT"`
	PrivateKeyPath  string `envconfig:"PRIVATE_KEY_PATH"`
	PublicKeyPath   string `envconfig:"PUBLIC_KEY_PATH"`
	JwtExpDelta     int    `envconfig:"JWT_EXPIRATION_DELTA"`
	JwtRefExpDelta  int    `envconfig:"JWT_REFRESH_EXPIRATION_DELTA"`
	StorageRootPath string `envconfig:"STORAGE_ROOT_PATH"`
}

// Config ...
type Config struct {
	Database DatabaseCfg
	Service  ServiceCfg
}

var cfg *Config

// Init ...
func Init() (*Config, error) {

	cfg = &Config{}

	if err := envconfig.Process("", cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Get ...
func Get() *Config {
	return cfg
}
