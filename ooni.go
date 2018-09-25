package ooni

import (
	"io/ioutil"
	"os"
	"path"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/config"
	"github.com/ooni/probe-cli/internal/bindata"
	"github.com/ooni/probe-cli/internal/database"
	"github.com/ooni/probe-cli/internal/legacy"
	"github.com/ooni/probe-cli/internal/onboard"
	"github.com/ooni/probe-cli/utils"
	"github.com/pkg/errors"
	"upper.io/db.v3/lib/sqlbuilder"
)

const Version = "3.0.0-alpha.2"

// Context for OONI Probe
type Context struct {
	Config   *config.Config
	DB       sqlbuilder.Database
	Location *utils.LocationInfo
	IsBatch  bool

	Home    string
	TempDir string

	dbPath     string
	configPath string
}

// MaybeLocationLookup will lookup the location of the user unless it's already cached
func (c *Context) MaybeLocationLookup() error {
	if c.Location == nil {
		return c.LocationLookup()
	}
	return nil
}

// LocationLookup lookup the location of the user via geoip
func (c *Context) LocationLookup() error {
	var err error

	if err = c.MaybeDownloadDataFiles(); err != nil {
		log.WithError(err).Error("failed to download data files")
	}

	geoipDir := utils.GeoIPDir(c.Home)
	c.Location, err = utils.GeoIPLookup(geoipDir)
	if err != nil {
		return err
	}

	return nil
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

// MaybeDownloadDataFiles will download geoip data files if they are not present
func (c *Context) MaybeDownloadDataFiles() error {
	geoipDir := utils.GeoIPDir(c.Home)
	if _, err := os.Stat(path.Join(geoipDir, "GeoLite2-Country.mmdb")); os.IsNotExist(err) {
		log.Debugf("Downloading GeoIP database files")
		if err := utils.DownloadGeoIPDatabaseFiles(geoipDir); err != nil {
			return err
		}
	}
	if _, err := os.Stat(path.Join(geoipDir, "GeoIP.dat")); os.IsNotExist(err) {
		log.Debugf("Downloading legacy GeoIP database Files")
		if err := utils.DownloadLegacyGeoIPDatabaseFiles(geoipDir); err != nil {
			return err
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

// NewContext instance.
func NewContext(configPath string, homePath string) *Context {
	return &Context{
		Home:       homePath,
		Config:     &config.Config{},
		configPath: configPath,
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
