package config

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/utils"
	"github.com/pkg/errors"
)

// ConfigVersion is the current version of the config
const ConfigVersion = 1

// ReadConfig reads the configuration from the path
func ReadConfig(path string) (*Config, error) {
	b, err := os.ReadFile(path) // #nosec G304 - this is working as intended
	if err != nil {
		return nil, err
	}

	c, err := ParseConfig(b)
	if err != nil {
		return nil, errors.Wrap(err, "parsing config")
	}
	c.path = path
	return c, err
}

// ParseConfig returns config from JSON bytes.
func ParseConfig(b []byte) (*Config, error) {
	var c Config

	if err := json.Unmarshal(b, &c); err != nil {
		return nil, errors.Wrap(err, "parsing json")
	}

	home, err := utils.GetOONIHome()
	if err != nil {
		return nil, err
	}
	c.path = utils.ConfigPath(home)

	return &c, nil
}

// Config for the OONI Probe installation
type Config struct {
	// Private settings
	Comment         string `json:"_"`
	Version         int64  `json:"_version"`
	InformedConsent bool   `json:"_informed_consent"`

	Sharing  Sharing  `json:"sharing"`
	Nettests Nettests `json:"nettests"`
	Advanced Advanced `json:"advanced"`

	mutex sync.Mutex
	path  string
}

// Write the config file in json to the path
func (c *Config) Write() error {
	c.Lock()
	defer c.Unlock()
	configJSON, _ := json.MarshalIndent(c, "", "  ")
	if c.path == "" {
		return errors.New("config file path is empty")
	}
	if err := os.WriteFile(c.path, configJSON, 0644); err != nil {
		return errors.Wrap(err, "writing config JSON")
	}
	return nil
}

// Lock acquires the write mutex
func (c *Config) Lock() {
	c.mutex.Lock()
}

// Unlock releases the write mutex
func (c *Config) Unlock() {
	c.mutex.Unlock()
}

// MaybeMigrate checks the current config version and the config file on disk
// and if necessary performs and upgrade of the configuration file.
func (c *Config) MaybeMigrate() error {
	if c.Version < ConfigVersion {
		return c.Write()
	}
	return nil
}
