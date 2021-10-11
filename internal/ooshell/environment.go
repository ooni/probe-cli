package ooshell

//
// environment.go
//
// Defines the Environment struct.
//

import (
	"context"
	"os"
	"path/filepath"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/version"
)

// Environ is the environment in which the shell runs. We recommend
// creating a new Environ using NewEnvironment and then changing the
// defaults configured by such a factory. Otherwise, keep in mind that
// every field marked as MANDATORY requires explicit initalization.
type Environ struct {
	// Annotations contains OPTIONAL annotations for measurements.
	Annotations []string

	// DB is the MANDATORY database in which to write results.
	DB DB

	// Categories OPTIONALLY allows you to restrict the categories
	// of URLs that you would like to test.
	Categories []string

	// Charging OPTIONALLY indicates whether we're not running on battery.
	Charging bool

	// Inputs contains OPTIONAL extra inouts for the experiment.
	Inputs []string

	// InputFilePaths contains OPTIONAL files from which to read extra inputs.
	InputFilePaths []string

	// KVStoreDir is the MANDATORY directory where to store key-value pairs.
	KVStoreDir string

	// Logger is the MANDATORY logger to use.
	Logger *log.Logger

	// MaxRuntime is the OPTIONAL maximum runtime in seconds.
	MaxRuntime int64

	// NoCollector OPTIONALLY allows disabling automatic measurement submission.
	NoCollector bool

	// NoJSON OPTIONALLY allows disabling automatically saving measurements.
	NoJSON bool

	// OnWiFi OPTIONALLY allows indicating whether we're using metered
	// or non-metered connections. True implies non-metered.
	OnWiFi bool

	// Options contains OPTIONAL extra options for the experiment.
	Options []string

	// ProbeServicesURL is the OPTIONAL probe-services URL.
	ProbeServicesURL string

	// ProxyURL is the OPTIONAL proxy URL. We use the proxy to
	// speak with the OONI backend services.
	//
	// There are four use cases:
	//
	// 1. if this field is empty, we won't use any proxy;
	//
	// 2. if this field is like `socks5://<address>:<port>`,
	// we use the given socks5 proxy;
	//
	// 3. if this field is `psiphon:///`, we create and
	// use a tunnel using the embedded psiphon;
	//
	// 4. if this field is `tor:///`, we attempt to
	// start tor and use its socks5 proxy.
	ProxyURL string

	// Random OPTIONALLY allows to randomly shuffle inputs.
	Random bool

	// ReportFile is the OPTIONAL name of the file where to save the report.
	ReportFile string

	// RunType is MANDATORY and MUST be one of "manual" and "timed".
	RunType string

	// SoftwareName is the name of this application (MANDATORY).
	SoftwareName string

	// SoftwareVersion is the version of this application (MANDATORY).
	SoftwareVersion string

	// TorArgs contains OPTIONAL arguments for the tor daemon.
	TorArgs []string

	// TorBinary is the OPTIONAL tor binary path.
	TorBinary string

	// TunnelDir is the MANDATORY directory where to cache private
	// information used by psiphon and tor tunnels.
	TunnelDir string
}

// NewEnvironment constructs a new environment using as root
// directory the given rootDir argument.
func NewEnvironment(rootDir string) *Environ {
	logger := NewLogger(os.Stderr)
	logger.Infof("root directory: %s", rootDir)
	return &Environ{
		Annotations:      []string{},
		Categories:       []string{},
		Charging:         true,
		DB:               NewLoggerDB(logger),
		Inputs:           []string{},
		InputFilePaths:   []string{},
		KVStoreDir:       filepath.Join(rootDir, "kvstore2"),
		Logger:           logger,
		MaxRuntime:       0,
		NoCollector:      false,
		NoJSON:           false,
		OnWiFi:           true,
		Options:          []string{},
		ProbeServicesURL: "",
		ProxyURL:         "",
		Random:           false,
		ReportFile:       "report.jsonl",
		RunType:          "manual",
		SoftwareName:     "miniooni",
		SoftwareVersion:  version.Version,
		TorArgs:          []string{},
		TorBinary:        "",
		TunnelDir:        filepath.Join(rootDir, "tunnel"),
	}
}

// RunExperiments runs the experiments with the given names inside a named group.
//
// Arguments:
//
// - ctx is the context for deadline/timeout/cancellation;
//
// - group is the arbitrary name of the group;
//
// - names is the list of experiment names inside this group.
//
// Returns nil on success, a non-nil error otherwise.
func (env *Environ) RunExperiments(
	ctx context.Context, group string, names ...string) error {
	sess, err := env.newSessionDB(ctx)
	if err != nil {
		return err
	}
	defer sess.Close()
	result, err := env.newResultDB(sess, group, names)
	if err != nil {
		return err
	}
	return result.Run(ctx)
}
