package ooni

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/openobservatory/gooni/config"
	"github.com/pkg/errors"
)

// GetOONIHome returns the path to the OONI Home
func GetOONIHome() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}

	path := filepath.Join(home, ".ooni")
	return path, nil
}

// Config for the OONI Probe installation
type Config struct {
	// Private settings
	Comment         string `json:"_"`
	ConfigVersion   string `json:"_config_version"`
	InformedConsent bool   `json:"_informed_consent"`

	AutoUpdate       bool                    `json:"auto_update"`
	Sharing          config.Sharing          `json:"sharing"`
	Notifications    config.Notifications    `json:"notifications"`
	AutomatedTesting config.AutomatedTesting `json:"automated_testing"`
	NettestGroups    config.NettestGroups    `json:"test_settings"`
	Advanced         config.Sharing          `json:"advanced"`
}

// Default config settings
func (c *Config) Default() error {
	return nil
}

// Validate the config file
func (c *Config) Validate() error {
	return nil
}

// ParseConfig returns config from JSON bytes.
func ParseConfig(b []byte) (*Config, error) {
	c := &Config{}

	if err := json.Unmarshal(b, c); err != nil {
		return nil, errors.Wrap(err, "parsing json")
	}

	if err := c.Default(); err != nil {
		return nil, errors.Wrap(err, "defaulting")
	}

	if err := c.Validate(); err != nil {
		return nil, errors.Wrap(err, "validating")
	}

	return c, nil
}

// ReadConfig reads the configuration from the path
func ReadConfig(path string) (*Config, error) {
	b, err := ioutil.ReadFile(path)

	if os.IsNotExist(err) {
		c := &Config{}

		if err = c.Default(); err != nil {
			return nil, errors.Wrap(err, "defaulting")
		}

		if err = c.Validate(); err != nil {
			return nil, errors.Wrap(err, "validating")
		}
	}

	if err != nil {
		return nil, errors.Wrap(err, "reading file")
	}

	return ParseConfig(b)
}
