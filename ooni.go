package ooni

import (
	"context"
	"io/ioutil"
	"os"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/config"
	"github.com/ooni/probe-cli/internal/bindata"
	"github.com/ooni/probe-cli/internal/database"
	"github.com/ooni/probe-cli/internal/enginex"
	"github.com/ooni/probe-cli/internal/legacy"
	"github.com/ooni/probe-cli/internal/onboard"
	"github.com/ooni/probe-cli/utils"
	"github.com/ooni/probe-cli/version"
	"github.com/ooni/probe-engine/session"
	"github.com/pkg/errors"
	"upper.io/db.v3/lib/sqlbuilder"
)

// Context for OONI Probe
type Context struct {
	Config  *config.Config
	DB      sqlbuilder.Database
	IsBatch bool
	Session *session.Session

	Home    string
	TempDir string

	dbPath     string
	configPath string
}

// MaybeLocationLookup will lookup the location of the user unless it's already cached
func (c *Context) MaybeLocationLookup() error {
	return c.Session.MaybeLookupLocation(context.Background())
}

// MaybeOnboarding will run the onboarding process only if the informed consent
// config option is set to false
func (c *Context) MaybeOnboarding() error {
	if c.Config.InformedConsent == false {
		if c.IsBatch == true {
			return errors.New("cannot run onboarding in batch mode")
		}
		if err := onboard.Onboarding(c.Config); err != nil {
			return errors.Wrap(err, "onboarding")
		}
	}
	return nil
}

// Init the OONI manager
func (c *Context) Init() error {
	var err error

	if err = legacy.MaybeMigrateHome(); err != nil {
		return errors.Wrap(err, "migrating home")
	}

	if err = MaybeInitializeHome(c.Home); err != nil {
		return err
	}

	if c.configPath != "" {
		log.Debugf("Reading config file from %s", c.configPath)
		c.Config, err = config.ReadConfig(c.configPath)
	} else {
		log.Debug("Reading default config file")
		c.Config, err = InitDefaultConfig(c.Home)
	}
	if err != nil {
		return err
	}

	c.dbPath = utils.DBDir(c.Home, "main")
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

// NewContext creates a new context instance.
func NewContext(configPath string, homePath string) *Context {
	return &Context{
		Home:       homePath,
		Config:     &config.Config{},
		configPath: configPath,
		Session: session.New(
			enginex.Logger,
			"ooniprobe-desktop",
			version.Version,
			utils.AssetsDir(homePath),
			nil, // explicit proxy url.URL
			nil, // explicit tls.Config
		),
	}
}

// MaybeInitializeHome does the setup for a new OONI Home
func MaybeInitializeHome(home string) error {
	for _, d := range utils.RequiredDirs(home) {
		if _, e := os.Stat(d); e != nil {
			if err := os.MkdirAll(d, 0700); err != nil {
				return err
			}
		}
	}
	return nil
}

// InitDefaultConfig reads the config from common locations or creates it if
// missing.
func InitDefaultConfig(home string) (*config.Config, error) {
	var (
		err        error
		c          *config.Config
		configPath = utils.ConfigPath(home)
	)

	c, err = config.ReadConfig(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Debugf("writing default config to %s", configPath)
			var data []byte
			data, err = bindata.Asset("data/default-config.json")
			if err != nil {
				return nil, err
			}
			err = ioutil.WriteFile(
				configPath,
				data,
				0644,
			)
			if err != nil {
				return nil, err
			}
			return InitDefaultConfig(home)
		}
		return nil, err
	}

	return c, nil
}
