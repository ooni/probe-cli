package ooni

import (
	"io/ioutil"
	"os"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/config"
	"github.com/ooni/probe-cli/internal/bindata"
	"github.com/ooni/probe-cli/internal/database"
	"github.com/ooni/probe-cli/internal/enginex"
	"github.com/ooni/probe-cli/internal/legacy"
	"github.com/ooni/probe-cli/utils"
	"github.com/ooni/probe-cli/version"
	engine "github.com/ooni/probe-engine"
	"github.com/pkg/errors"
	"upper.io/db.v3/lib/sqlbuilder"
)

// Context for OONI Probe
type Context struct {
	Config       *config.Config
	DB           sqlbuilder.Database
	IsBatch      bool
	IsTerminated bool
	Session      *engine.Session

	Home    string
	TempDir string

	dbPath     string
	configPath string
}

// MaybeLocationLookup will lookup the location of the user unless it's already cached
func (c *Context) MaybeLocationLookup() error {
	return c.Session.MaybeLookupLocation()
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
	if err = c.Config.MaybeMigrate(); err != nil {
		return errors.Wrap(err, "migrating config")
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

	sess, err := engine.NewSession(engine.SessionConfig{
		Logger:          enginex.Logger,
		SoftwareName:    "ooniprobe-desktop",
		SoftwareVersion: version.Version,
		AssetsDir:       utils.AssetsDir(c.Home),
		TempDir:         c.TempDir,
	})
	if err != nil {
		return err
	}
	c.Session = sess

	return nil
}

// NewContext creates a new context instance.
func NewContext(configPath string, homePath string) *Context {
	return &Context{
		Home:         homePath,
		Config:       &config.Config{},
		configPath:   configPath,
		IsTerminated: false,
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
