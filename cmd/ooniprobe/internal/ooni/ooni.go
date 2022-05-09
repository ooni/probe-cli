package ooni

import (
	"context"
	_ "embed" // because we embed a file
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/config"
	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/database"
	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/enginex"
	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/utils"
	"github.com/ooni/probe-cli/v3/internal/atomicx"
	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/engine/legacy/assetsdir"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/pkg/errors"
	"github.com/upper/db/v4"
)

// DefaultSoftwareName is the default software name.
const DefaultSoftwareName = "ooniprobe-cli"

// ProbeCLI is the OONI Probe CLI context.
type ProbeCLI interface {
	Config() *config.Config
	DB() db.Session
	IsBatch() bool
	Home() string
	TempDir() string
	NewProbeEngine(ctx context.Context, runType model.RunType) (ProbeEngine, error)
}

// ProbeEngine is an instance of the OONI Probe engine.
type ProbeEngine interface {
	Close() error
	MaybeLookupLocation() error
	ProbeASNString() string
	ProbeCC() string
	ProbeIP() string
	ProbeNetworkName() string
}

// Probe contains the ooniprobe CLI context.
type Probe struct {
	config  *config.Config
	db      db.Session
	isBatch bool

	home      string
	tempDir   string
	tunnelDir string

	dbPath     string
	configPath string

	isTerminated *atomicx.Int64

	softwareName    string
	softwareVersion string
}

// SetIsBatch sets the value of isBatch.
func (p *Probe) SetIsBatch(v bool) {
	p.isBatch = v
}

// IsBatch returns whether we're running in batch mode.
func (p *Probe) IsBatch() bool {
	return p.isBatch
}

// Config returns the configuration
func (p *Probe) Config() *config.Config {
	return p.config
}

// DB returns the database we're using
func (p *Probe) DB() db.Session {
	return p.db
}

// Home returns the home directory.
func (p *Probe) Home() string {
	return p.home
}

// TempDir returns the temporary directory.
func (p *Probe) TempDir() string {
	return p.tempDir
}

// IsTerminated checks to see if the isTerminatedAtomicInt is set to a non zero
// value and therefore we have received the signal to shutdown the running test
func (p *Probe) IsTerminated() bool {
	return p.isTerminated.Load() != 0
}

// Terminate interrupts the running context
func (p *Probe) Terminate() {
	p.isTerminated.Add(1)
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

	if err = MaybeInitializeHome(p.home); err != nil {
		return err
	}

	if p.configPath != "" {
		log.Debugf("Reading config file from %s", p.configPath)
		p.config, err = config.ReadConfig(p.configPath)
	} else {
		log.Debug("Reading default config file")
		p.config, err = InitDefaultConfig(p.home)
	}
	if err != nil {
		return err
	}
	if err = p.config.MaybeMigrate(); err != nil {
		return errors.Wrap(err, "migrating config")
	}

	p.dbPath = utils.DBDir(p.home, "main")
	log.Debugf("Connecting to database sqlite3://%s", p.dbPath)
	db, err := database.Connect(p.dbPath)
	if err != nil {
		return err
	}
	p.db = db

	// We cleanup the assets files used by versions of ooniprobe
	// older than v3.9.0, where we started embedding the assets
	// into the binary and use that directly. This cleanup doesn't
	// remove the whole directory but only known files inside it
	// and then the directory itself, if empty. We explicitly discard
	// the return value as it does not matter to us here.
	_, _ = assetsdir.Cleanup(utils.AssetsDir(p.home))

	tempDir, err := ioutil.TempDir("", "ooni")
	if err != nil {
		return errors.Wrap(err, "creating TempDir")
	}
	p.tempDir = tempDir
	p.tunnelDir = utils.TunnelDir(p.home)

	p.softwareName = softwareName
	p.softwareVersion = softwareVersion
	return nil
}

// NewSession creates a new ooni/probe-engine session using the
// current configuration inside the context. The caller must close
// the session when done using it, by calling sess.Close().
func (p *Probe) NewSession(ctx context.Context, runType model.RunType) (*engine.Session, error) {
	kvstore, err := kvstore.NewFS(
		utils.EngineDir(p.home),
	)
	if err != nil {
		return nil, errors.Wrap(err, "creating engine's kvstore")
	}
	if err := os.MkdirAll(p.tunnelDir, 0700); err != nil {
		return nil, errors.Wrap(err, "creating tunnel dir")
	}
	// When the software name is the default software name and we're running
	// in unattended mode, adjust the software name accordingly.
	//
	// See https://github.com/ooni/probe/issues/2081.
	softwareName := p.softwareName
	if runType == model.RunTypeTimed && softwareName == DefaultSoftwareName {
		softwareName = DefaultSoftwareName + "-unattended"
	}
	return engine.NewSession(ctx, engine.SessionConfig{
		KVStore:         kvstore,
		Logger:          enginex.Logger,
		SoftwareName:    softwareName,
		SoftwareVersion: p.softwareVersion,
		TempDir:         p.tempDir,
		TunnelDir:       p.tunnelDir,
	})
}

// NewProbeEngine creates a new ProbeEngine instance.
func (p *Probe) NewProbeEngine(ctx context.Context, runType model.RunType) (ProbeEngine, error) {
	sess, err := p.NewSession(ctx, runType)
	if err != nil {
		return nil, err
	}
	return sess, nil
}

// NewProbe creates a new probe instance.
func NewProbe(configPath string, homePath string) *Probe {
	return &Probe{
		home:         homePath,
		config:       &config.Config{},
		configPath:   configPath,
		isTerminated: &atomicx.Int64{},
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

//go:embed default-config.json
var defaultConfig []byte

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
			if err = os.WriteFile(configPath, defaultConfig, 0644); err != nil {
				return nil, err
			}
			// If the user did the informed consent procedure in
			// probe-legacy, migrate it over.
			if utils.DidLegacyInformedConsent() {
				c, err := config.ReadConfig(configPath)
				if err != nil {
					return nil, err
				}
				c.Lock()
				c.InformedConsent = true
				c.Unlock()
				if err := c.Write(); err != nil {
					return nil, err
				}
			}

			return InitDefaultConfig(home)
		}
		return nil, err
	}

	return c, nil
}
