// Package middleware implements the TrackerLogic interface by executing
// a series of middleware hooks.
package middleware

import (
	"errors"
	"sync"

	"gopkg.in/yaml.v2"
)

var (
	driversM sync.RWMutex
	drivers  = make(map[string]Driver)

	// ErrDriverDoesNotExist is the error returned by NewMiddleware when a
	// middleware driver with that name does not exist.
	ErrDriverDoesNotExist = errors.New("middleware driver with that name does not exist")
)

// Driver is the interface used to initialize a new type of middleware.
//
// The options parameter is YAML encoded bytes that should be unmarshalled into
// the hook's custom configuration.
type Driver interface {
	NewHook(options []byte) (Hook, error)
}

// RegisterDriver makes a Driver available by the provided name.
//
// If called twice with the same name, the name is blank, or if the provided
// Driver is nil, this function panics.
func RegisterDriver(name string, d Driver) {
	if name == "" {
		panic("middleware: could not register a Driver with an empty name")
	}
	if d == nil {
		panic("middleware: could not register a nil Driver")
	}

	driversM.Lock()
	defer driversM.Unlock()

	if _, dup := drivers[name]; dup {
		panic("middleware: RegisterDriver called twice for " + name)
	}

	drivers[name] = d
}

// New attempts to initialize a new middleware instance from the
// list of registered Drivers.
//
// If a driver does not exist, returns ErrDriverDoesNotExist.
func New(name string, optionBytes []byte) (Hook, error) {
	driversM.RLock()
	defer driversM.RUnlock()

	var d Driver
	d, ok := drivers[name]
	if !ok {
		return nil, ErrDriverDoesNotExist
	}

	return d.NewHook(optionBytes)
}

// HookConfig is the generic configuration format used for all registered Hooks.
type HookConfig struct {
	Name    string                 `yaml:"name"`
	Options map[string]interface{} `yaml:"options"`
}

// HooksFromHookConfigs is a utility function for initializing Hooks in bulk.
func HooksFromHookConfigs(cfgs []HookConfig) (hooks []Hook, err error) {
	for _, cfg := range cfgs {
		// Marshal the options back into bytes.
		var optionBytes []byte
		optionBytes, err = yaml.Marshal(cfg.Options)
		if err != nil {
			return
		}

		var h Hook
		h, err = New(cfg.Name, optionBytes)
		if err != nil {
			return
		}

		hooks = append(hooks, h)
	}

	return
}
