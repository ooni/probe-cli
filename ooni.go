package ooni

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/apex/log"
	"github.com/jmoiron/sqlx"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/openobservatory/gooni/config"
	"github.com/openobservatory/gooni/internal/database"
	"github.com/openobservatory/gooni/internal/legacy"
	"github.com/pkg/errors"
)

// Onboarding process
func Onboarding(c *Config) error {
	log.Info("Onboarding starting")

	// To prevent races we always must acquire the config file lock before
	// changing it.
	c.Lock()
	c.InformedConsent = true
	c.Unlock()

	if err := c.Write(); err != nil {
		log.Warnf("Failed to save informed consent: %v", err)
		return err
	}
	return nil
}

// Context for OONI Probe
type Context struct {
	Config *Config
	DB     *sqlx.DB

	Home       string
	TempDir    string
	dbPath     string
	configPath string
}

// Init the OONI manager
func (c *Context) Init() error {
	var err error

	if err = legacy.MaybeMigrateHome(); err != nil {
		return errors.Wrap(err, "migrating home")
	}

	if err = CreateHomeDirs(c.Home); err != nil {
		return err
	}

	if c.configPath != "" {
		log.Debugf("Reading config file from %s", c.configPath)
		c.Config, err = ReadConfig(c.configPath)
	} else {
		log.Debug("Reading default config file")
		c.Config, err = ReadDefaultConfigPaths(c.Home)
	}
	if err != nil {
		return err
	}

	c.dbPath = filepath.Join(c.Home, "db", "main.sqlite3")
	if c.Config.InformedConsent == false {
		if err = Onboarding(c.Config); err != nil {
			return errors.Wrap(err, "onboarding")
		}
	}

	log.Debugf("Connecting to database sqlite3://%s", c.dbPath)
	db, err := database.Connect(c.dbPath)
	if err != nil {
		return err
	}
	c.DB = db

	tempDir, err := ioutil.TempDir("", "ooni")
	if err != nil {
		return errors.Wrap(err, "creating TempDir")
	}
	c.TempDir = tempDir

	return nil
}

// NewContext instance.
func NewContext(configPath string, homePath string) *Context {
	return &Context{
		Home:       homePath,
		Config:     &Config{},
		configPath: configPath,
	}
}

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
	Advanced         config.Advanced         `json:"advanced"`

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
	home, err := GetOONIHome()
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

// CreateHomeDirs creates the OONI home subdirectories
func CreateHomeDirs(home string) error {
	requiredDirs := []string{"db", "msmts", "geoip"}
	for _, d := range requiredDirs {
		if _, e := os.Stat(filepath.Join(home, d)); e != nil {
			if err := os.MkdirAll(filepath.Join(home, d), 0700); err != nil {
				return err
			}
		}
	}

	return nil
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

		return c, nil
	}

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

// ReadDefaultConfigPaths from common locations.
func ReadDefaultConfigPaths(home string) (*Config, error) {
	var paths = []string{
		filepath.Join(home, "config.json"),
	}
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			c, err := ReadConfig(path)
			if err != nil {
				return nil, err
			}
			return c, nil
		}
	}

	// Run from the default config
	return ReadConfig(paths[0])
}
