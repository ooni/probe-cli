// Command miniooni is a simple binary for research and QA purposes
// with a CLI interface similar to MK and OONI Probe v2.x.
package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/humanize"
	"github.com/ooni/probe-cli/v3/internal/legacy/assetsdir"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/version"
	"github.com/pborman/getopt/v2"
)

// Options contains the options you can set from the CLI.
type Options struct {
	Annotations      []string
	ExtraOptions     []string
	HomeDir          string
	Inputs           []string
	InputFilePaths   []string
	MaxRuntime       int64
	NoJSON           bool
	NoCollector      bool
	ProbeServicesURL string
	Proxy            string
	Random           bool
	RepeatEvery      int64
	ReportFile       string
	TorArgs          []string
	TorBinary        string
	Tunnel           string
	Verbose          bool
	Version          bool
	Yes              bool
}

var globalOptions Options

func init() {
	getopt.FlagLong(
		&globalOptions.Annotations, "annotation", 'A', "Add annotaton", "KEY=VALUE",
	)
	getopt.FlagLong(
		&globalOptions.ExtraOptions, "option", 'O',
		"Pass an option to the experiment", "KEY=VALUE",
	)
	getopt.FlagLong(
		&globalOptions.InputFilePaths, "input-file", 'f',
		"Path to input file to supply test-dependent input. File must contain one input per line.", "PATH",
	)
	getopt.FlagLong(
		&globalOptions.HomeDir, "home", 0,
		"Force specific home directory", "PATH",
	)
	getopt.FlagLong(
		&globalOptions.Inputs, "input", 'i',
		"Add test-dependent input to the test input", "INPUT",
	)
	getopt.FlagLong(
		&globalOptions.MaxRuntime, "max-runtime", 0,
		"Maximum runtime in seconds when looping over a list of inputs (zero means infinite)", "N",
	)
	getopt.FlagLong(
		&globalOptions.NoJSON, "no-json", 'N', "Disable writing to disk",
	)
	getopt.FlagLong(
		&globalOptions.NoCollector, "no-collector", 'n', "Don't use a collector",
	)
	getopt.FlagLong(
		&globalOptions.ProbeServicesURL, "probe-services", 0,
		"Set the URL of the probe-services instance you want to use", "URL",
	)
	getopt.FlagLong(
		&globalOptions.Proxy, "proxy", 0, "Set the proxy URL", "URL",
	)
	getopt.FlagLong(
		&globalOptions.Random, "random", 0, "Randomize inputs",
	)
	getopt.FlagLong(
		&globalOptions.RepeatEvery, "repeat-every", 0,
		"Repeat the measurement every INTERVAL number of seconds", "INTERVAL",
	)
	getopt.FlagLong(
		&globalOptions.ReportFile, "reportfile", 'o',
		"Set the report file path", "PATH",
	)
	getopt.FlagLong(
		&globalOptions.TorArgs, "tor-args", 0,
		"Extra args for tor binary (may be specified multiple times)",
	)
	getopt.FlagLong(
		&globalOptions.TorBinary, "tor-binary", 0,
		"Specify path to a specific tor binary",
	)
	getopt.FlagLong(
		&globalOptions.Tunnel, "tunnel", 0,
		"Name of the tunnel to use (one of `tor`, `psiphon`)",
	)
	getopt.FlagLong(
		&globalOptions.Verbose, "verbose", 'v', "Increase verbosity",
	)
	getopt.FlagLong(
		&globalOptions.Version, "version", 0, "Print version and exit",
	)
	getopt.FlagLong(
		&globalOptions.Yes, "yes", 'y',
		"Assume yes as the answer to all questions",
	)
}

// main is the main function of miniooni. This function parses the command line
// options and uses a global state. Use MainWithConfiguration if you want to avoid
// using any global state and relying on command line options.
//
// This function will panic in case of a fatal error. It is up to you that
// integrate this function to either handle the panic of ignore it.
func main() {
	getopt.Parse()
	if globalOptions.Version {
		fmt.Printf("%s\n", version.Version)
		os.Exit(0)
	}
	runtimex.PanicIfFalse(len(getopt.Args()) == 1, "Missing experiment name")
	runtimex.PanicOnError(engine.CheckEmbeddedPsiphonConfig(), "Invalid embedded psiphon config")
	MainWithConfiguration(getopt.Arg(0), globalOptions)
}

// tunnelAndProxy is the text printed when the user specifies
// both the --tunnel and the --proxy options
const tunnelAndProxy = `USAGE ERROR: The --tunnel option and the --proxy
option cannot be specified at the same time. The --tunnel option is actually
just syntactic sugar for --proxy. Setting --tunnel=psiphon is currently the
equivalent of setting --proxy=psiphon:///. This MAY change in a future version
of miniooni, when we will allow a tunnel to use a proxy.
`

// MainWithConfiguration is the miniooni main with a specific configuration
// represented by the experiment name and the current options.
//
// This function will panic in case of a fatal error. It is up to you that
// integrate this function to either handle the panic of ignore it.
func MainWithConfiguration(experimentName string, currentOptions Options) {
	runtimex.PanicIfTrue(currentOptions.Proxy != "" && currentOptions.Tunnel != "",
		tunnelAndProxy)
	if currentOptions.Tunnel != "" {
		currentOptions.Proxy = fmt.Sprintf("%s:///", currentOptions.Tunnel)
	}

	logger := &log.Logger{Level: log.InfoLevel, Handler: &logHandler{Writer: os.Stderr}}
	if currentOptions.Verbose {
		logger.Level = log.DebugLevel
	}
	if currentOptions.ReportFile == "" {
		currentOptions.ReportFile = "report.jsonl"
	}
	log.Log = logger
	for {
		mainSingleIteration(logger, experimentName, currentOptions)
		if currentOptions.RepeatEvery <= 0 {
			break
		}
		log.Infof("waiting %ds before repeating the measurement", currentOptions.RepeatEvery)
		log.Info("use Ctrl-C to interrupt miniooni")
		time.Sleep(time.Duration(currentOptions.RepeatEvery) * time.Second)
	}
}

// mainSingleIteration runs a single iteration. There may be multiple iterations
// when the user specifies the --repeat-every command line flag.
func mainSingleIteration(logger model.Logger, experimentName string, currentOptions Options) {
	extraOptions := mustMakeMapStringAny(currentOptions.ExtraOptions)
	annotations := mustMakeMapStringString(currentOptions.Annotations)

	ctx := context.Background()

	//Mon Jan 2 15:04:05 -0700 MST 2006
	log.Infof("Current time: %s", time.Now().Format("2006-01-02 15:04:05 MST"))

	homeDir := gethomedir(currentOptions.HomeDir)
	runtimex.PanicIfFalse(homeDir != "", "home directory is empty")
	miniooniDir := path.Join(homeDir, ".miniooni")
	err := os.MkdirAll(miniooniDir, 0700)
	runtimex.PanicOnError(err, "cannot create $HOME/.miniooni directory")

	// We cleanup the assets files used by versions of ooniprobe
	// older than v3.9.0, where we started embedding the assets
	// into the binary and use that directly. This cleanup doesn't
	// remove the whole directory but only known files inside it
	// and then the directory itself, if empty. We explicitly discard
	// the return value as it does not matter to us here.
	assetsDir := path.Join(miniooniDir, "assets")
	_, _ = assetsdir.Cleanup(assetsDir)

	log.Debugf("miniooni state directory: %s", miniooniDir)
	log.Info("miniooni home directory: $HOME/.miniooni")

	acquireUserConsent(miniooniDir, currentOptions)

	sess := newSessionOrPanic(ctx, currentOptions, miniooniDir, logger)
	defer func() {
		sess.Close()
		log.Infof("whole session: recv %s, sent %s",
			humanize.SI(sess.KibiBytesReceived()*1024, "byte"),
			humanize.SI(sess.KibiBytesSent()*1024, "byte"),
		)
	}()
	lookupBackendsOrPanic(ctx, sess)
	lookupLocationOrPanic(ctx, sess)

	// We handle the oonirun experiment name specially. The user must specify
	// `miniooni -i {OONIRunURL} oonirun` to run a OONI Run URL (v1 or v2).
	if experimentName == "oonirun" {
		ooniRunMain(ctx, sess, currentOptions, annotations)
		return
	}

	// Otherwise just run OONI experiments as we normally do.
	runx(ctx, sess, experimentName, annotations, extraOptions, currentOptions)
}
