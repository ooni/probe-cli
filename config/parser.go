package config

import (
	"encoding/json"
	"io/ioutil"
	"sync"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/internal/crashreport"
	"github.com/ooni/probe-cli/utils"
	"github.com/pkg/errors"
)

// ReadConfig reads the configuration from the path
func ReadConfig(path string) (*Config, error) {
	b, err := ioutil.ReadFile(path)
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

	if err := c.Default(); err != nil {
		return nil, errors.Wrap(err, "defaulting")
	}

	if err := c.Validate(); err != nil {
		return nil, errors.Wrap(err, "validating")
	}
	if c.Advanced.SendCrashReports == false {
		log.Info("Disabling crash reporting.")
		crashreport.Disabled = true
	}

	return &c, nil
}

// Config for the OONI Probe installation
type Config struct {
	// Private settings
	Comment         string `json:"_"`
	Version         int64  `json:"_version"`
	InformedConsent bool   `json:"_informed_consent"`
	IsBeta          bool   `json:"_is_beta"` // This is a boolean flag used to indicate this installation of OONI Probe was a beta install. These installations will have their data deleted across releases.

	Sharing  Sharing  `json:"sharing"`
	Advanced Advanced `json:"advanced"`

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

	c.path = utils.ConfigPath(home)
	return nil
}

// Validate the config file
func (c *Config) Validate() error {
	return nil
}
