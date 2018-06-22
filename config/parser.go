package config

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"sync"

	"github.com/ooni/probe-cli/utils"
	"github.com/pkg/errors"
)

// ReadConfig reads the configuration from the path
func ReadConfig(path string) (*Config, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "reading file")
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

	if err := c.Default(); err != nil {
		return nil, errors.Wrap(err, "defaulting")
	}

	if err := c.Validate(); err != nil {
		return nil, errors.Wrap(err, "validating")
	}

	return &c, nil
}

// Config for the OONI Probe installation
type Config struct {
	// Private settings
	Comment         string `json:"_"`
	Version         int64  `json:"_version"`
	InformedConsent bool   `json:"_informed_consent"`

	AutoUpdate       bool             `json:"auto_update"`
	Sharing          Sharing          `json:"sharing"`
	Notifications    Notifications    `json:"notifications"`
	AutomatedTesting AutomatedTesting `json:"automated_testing"`
	NettestGroups    NettestGroups    `json:"test_settings"`
	Advanced         Advanced         `json:"advanced"`

	mutex sync.Mutex
	path  string
}

// Write the config file in json to the path
func (c *Config) Write() error {
	c.Lock()
	configJSON, _ := json.MarshalIndent(c, "", "  ")
	if c.path == "" {
		return errors.New("config file path is empty")
	}
	if err := ioutil.WriteFile(c.path, configJSON, 0644); err != nil {
		return errors.Wrap(err, "writing config JSON")
	}
	c.Unlock()
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

// Default config settings
func (c *Config) Default() error {
	home, err := utils.GetOONIHome()
	if err != nil {
		return err
	}

	c.path = filepath.Join(home, "config.json")
	return nil
}

// Validate the config file
func (c *Config) Validate() error {
	return nil
}
