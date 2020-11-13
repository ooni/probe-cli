package ooni

import (
	"io/ioutil"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/internal/bindata"
	"github.com/ooni/probe-cli/internal/config"
	"github.com/ooni/probe-cli/internal/database"
	"github.com/ooni/probe-cli/internal/enginex"
	"github.com/ooni/probe-cli/internal/utils"
	engine "github.com/ooni/probe-engine"
	"github.com/ooni/probe-engine/model"
	"github.com/pkg/errors"
	"upper.io/db.v3/lib/sqlbuilder"
)

// Probe contains the ooniprobe CLI context.
type Probe struct {
	Config  *config.Config
	DB      sqlbuilder.Database
	IsBatch bool

	Home    string
	TempDir string

	dbPath     string
	configPath string

	// We need to use a int32 in order to use the atomic.AddInt32/LoadInt32
	// operations to ensure consistent reads of the variables. We do not use
	// a 64 bit integer here because that may lead to crashes with 32 bit
	// OSes as documented in https://golang.org/pkg/sync/atomic/#pkg-note-BUG.
	isTerminatedAtomicInt int32

	softwareName    string
	softwareVersion string
}

// IsTerminated checks to see if the isTerminatedAtomicInt is set to a non zero
// value and therefore we have received the signal to shutdown the running test
func (p *Probe) IsTerminated() bool {
	i := atomic.LoadInt32(&p.isTerminatedAtomicInt)
	return i != 0
}

// Terminate interrupts the running context
func (p *Probe) Terminate() {
	atomic.AddInt32(&p.isTerminatedAtomicInt, 1)
}

// ListenForSignals will listen for SIGINT and SIGTERM. When it receives those
// signals it will set isTerminatedAtomicInt to non-zero, which will cleanly
// shutdown the test logic.
//
// TODO refactor this to use a cancellable context.Context instead of a bool
// flag, probably as part of: https://github.com/ooni/probe-cli/issues/45
func (p *Probe) ListenForSignals() {
	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-s
		log.Info("caught a stop signal, shutting down cleanly")
		p.Terminate()
	}()
}

// MaybeListenForStdinClosed will treat any error on stdin just
// like SIGTERM if and only if
//
//     os.Getenv("OONI_STDIN_EOF_IMPLIES_SIGTERM") == "true"
//
// When this feature is enabled, a collateral effect is that we swallow
// whatever is passed to us on the standard input.
//
// See https://github.com/ooni/probe-cli/pull/111 for more info
// regarding the design of this functionality.
//
// TODO refactor this to use a cancellable context.Context instead of a bool
// flag, probably as part of: https://github.com/ooni/probe-cli/issues/45
func (p *Probe) MaybeListenForStdinClosed() {
	if os.Getenv("OONI_STDIN_EOF_IMPLIES_SIGTERM") != "true" {
		return
	}
	go func() {
		defer p.Terminate()
		defer log.Info("stdin closed, shutting down cleanly")
		b := make([]byte, 1<<10)
		for {
			if _, err := os.Stdin.Read(b); err != nil {
				return
			}
		}
	}()
}

// Init the OONI manager
func (p *Probe) Init(softwareName, softwareVersion string) error {
	var err error

	if err = MaybeInitializeHome(p.Home); err != nil {
		return err
	}

	if p.configPath != "" {
		log.Debugf("Reading config file from %s", p.configPath)
		p.Config, err = config.ReadConfig(p.configPath)
	} else {
		log.Debug("Reading default config file")
		p.Config, err = InitDefaultConfig(p.Home)
	}
	if err != nil {
		return err
	}
	if err = p.Config.MaybeMigrate(); err != nil {
		return errors.Wrap(err, "migrating config")
	}

	p.dbPath = utils.DBDir(p.Home, "main")
	log.Debugf("Connecting to database sqlite3://%s", p.dbPath)
	db, err := database.Connect(p.dbPath)
	if err != nil {
		return err
	}
	p.DB = db

	tempDir, err := ioutil.TempDir("", "ooni")
	if err != nil {
		return errors.Wrap(err, "creating TempDir")
	}
	p.TempDir = tempDir

	p.softwareName = softwareName
	p.softwareVersion = softwareVersion
	return nil
}

// NewSession creates a new ooni/probe-engine session using the
// current configuration inside the context. The caller must close
// the session when done using it, by calling sess.Close().
func (p *Probe) NewSession() (*engine.Session, error) {
	kvstore, err := engine.NewFileSystemKVStore(
		utils.EngineDir(p.Home),
	)
	if err != nil {
		return nil, errors.Wrap(err, "creating engine's kvstore")
	}
	return engine.NewSession(engine.SessionConfig{
		AssetsDir: utils.AssetsDir(p.Home),
		KVStore:   kvstore,
		Logger:    enginex.Logger,
		PrivacySettings: model.PrivacySettings{
			IncludeASN:     p.Config.Sharing.IncludeASN,
			IncludeCountry: true,
			IncludeIP:      p.Config.Sharing.IncludeIP,
		},
		SoftwareName:    p.softwareName,
		SoftwareVersion: p.softwareVersion,
		TempDir:         p.TempDir,
	})
}

// NewProbe creates a new probe instance.
func NewProbe(configPath string, homePath string) *Probe {
	return &Probe{
		Home:                  homePath,
		Config:                &config.Config{},
		configPath:            configPath,
		isTerminatedAtomicInt: 0,
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
