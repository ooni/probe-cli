package ooni

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/openobservatory/gooni/config"
	"github.com/pkg/errors"
)

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
	// This is the default configuration:
	/*
			_: 'This is your OONI Probe config file. See https://ooni.io/help/ooniprobe-cli for help',
		auto_update: true,
		sharing: {
			include_ip: false,
			include_asn: true,
			include_gps: true,
			upload_results: true,
			send_crash_reports: true
		},
		notifications: {
			enabled: true,
			notify_on_test_completion: true,
			notify_on_news: false
		},
		automated_testing: {
			enabled: false,
			enabled_tests: [
				'web-connectivity',
				'facebook-messenger',
				'whatsapp',
				'telegram',
				'dash',
				'ndt',
				'http-invalid-request-line',
				'http-header-field-manipulation'
			],
			monthly_allowance: '300MB'
		},
		test_settings: {
			websites: {
				enabled_categories: []
			},
			instant_messaging: {
				enabled_tests: [
					'facebook-messenger',
					'whatsapp',
					'telegram'
				]
			},
			performance: {
				enabled_tests: [
					'ndt'
				],
				ndt_server: 'auto',
				ndt_server_port: 'auto',
				dash_server: 'auto',
				dash_server_port: 'auto'
			},
			middlebox: {
				enabled_tests: [
					'http-invalid-request-line',
					'http-header-field-manipulation'
				]
			}
		},
		advanced: {
			include_country: true,
			use_domain_fronting: true
		},
		_config_version: CONFIG_VERSION,
		_informed_consent: false
	*/
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
