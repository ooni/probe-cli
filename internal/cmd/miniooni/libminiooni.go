package main

//
// Core implementation
//

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/humanize"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/legacy/assetsdir"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/oonirun"
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
	RepeatEvery      time.Duration
	ReportFile       string
	TorArgs          []string
	TorBinary        string
	Tunnel           string
	Verbose          bool
	Version          bool
	Yes              bool
}

const (
	softwareName    = "miniooni"
	softwareVersion = version.Version
)

var (
	globalOptions Options
	startTime     = time.Now()
)

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
		"Repeat the measurement every INTERVAL (e.g., 30m, 1h, 2h)", "INTERVAL",
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
		"Accept risk of running OONI, new OONI Run links or OONI Run links modified upstream",
	)
}

func fatalIfFalse(cond bool, msg string) {
	if !cond {
		panic(msg)
	}
}

func fatalIfTrue(cond bool, msg string) {
	fatalIfFalse(!cond, msg)
}

// Main is the main function of miniooni. This function parses the command line
// options and uses a global state. Use MainWithConfiguration if you want to avoid
// using any global state and relying on command line options.
//
// This function will panic in case of a fatal error. It is up to you that
// integrate this function to either handle the panic of ignore it.
func Main() {
	getopt.Parse()
	if globalOptions.Version {
		fmt.Printf("%s\n", version.Version)
		os.Exit(0)
	}
	fatalIfFalse(len(getopt.Args()) == 1, "Missing experiment name")
	fatalOnError(engine.CheckEmbeddedPsiphonConfig(), "Invalid embedded psiphon config")
	MainWithConfiguration(getopt.Arg(0), globalOptions)
}

func split(s string) (string, string, error) {
	v := strings.SplitN(s, "=", 2)
	if len(v) != 2 {
		return "", "", errors.New("invalid key-value pair")
	}
	return v[0], v[1], nil
}

func fatalOnError(err error, msg string) {
	if err != nil {
		log.WithError(err).Warn(msg)
		panic(msg)
	}
}

func mustMakeMapString(input []string) (output map[string]string) {
	output = make(map[string]string)
	for _, opt := range input {
		key, value, err := split(opt)
		fatalOnError(err, "cannot split key-value pair")
		output[key] = value
	}
	return
}

func mustMakeMapAny(input []string) (output map[string]any) {
	output = make(map[string]any)
	for _, opt := range input {
		key, value, err := split(opt)
		fatalOnError(err, "cannot split key-value pair")
		output[key] = value
	}
	return
}

func mustParseURL(URL string) *url.URL {
	rv, err := url.Parse(URL)
	fatalOnError(err, "cannot parse URL")
	return rv
}

type logHandler struct {
	io.Writer
}

func (h *logHandler) HandleLog(e *log.Entry) (err error) {
	s := fmt.Sprintf("[%14.6f] <%s> %s", time.Since(startTime).Seconds(), e.Level, e.Message)
	if len(e.Fields) > 0 {
		s += fmt.Sprintf(": %+v", e.Fields)
	}
	s += "\n"
	_, err = h.Writer.Write([]byte(s))
	return
}

// See https://gist.github.com/miguelmota/f30a04a6d64bd52d7ab59ea8d95e54da
func gethomedir(optionsHome string) string {
	if optionsHome != "" {
		return optionsHome
	}
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	}
	if runtime.GOOS == "linux" {
		home := os.Getenv("XDG_CONFIG_HOME")
		if home != "" {
			return home
		}
		// fallthrough
	}
	return os.Getenv("HOME")
}

const riskOfRunningOONI = `
Do you consent to OONI Probe data collection?

OONI Probe collects evidence of internet censorship and measures
network performance:

- OONI Probe will likely test objectionable sites and services;

- Anyone monitoring your internet activity (such as a government
or Internet provider) may be able to tell that you are using OONI Probe;

- The network data you collect will be published automatically
unless you use miniooni's -n command line flag.

To learn more, see https://ooni.org/about/risks/.

If you're onboard, re-run the same command and add the --yes flag, to
indicate that you understand the risks. This will create an empty file
named 'consent' in $HOME/.miniooni, meaning that we know you opted in
and we will not ask you this question again.

`

func canOpen(filepath string) bool {
	stat, err := os.Stat(filepath)
	return err == nil && stat.Mode().IsRegular()
}

func maybeWriteConsentFile(yes bool, filepath string) (err error) {
	if yes {
		err = os.WriteFile(filepath, []byte("\n"), 0644)
	}
	return
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
	fatalIfTrue(currentOptions.Proxy != "" && currentOptions.Tunnel != "",
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
		log.Infof("waiting %s before repeating the measurement", currentOptions.RepeatEvery)
		log.Info("use Ctrl-C to interrupt miniooni")
		time.Sleep(currentOptions.RepeatEvery)
	}
}

// mainSingleIteration runs a single iteration. There may be multiple iterations
// when the user specifies the --repeat-every command line flag.
func mainSingleIteration(logger model.Logger, experimentName string, currentOptions Options) {
	extraOptions := mustMakeMapAny(currentOptions.ExtraOptions)
	annotations := mustMakeMapString(currentOptions.Annotations)

	ctx := context.Background()

	//Mon Jan 2 15:04:05 -0700 MST 2006
	log.Infof("Current time: %s", time.Now().Format("2006-01-02 15:04:05 MST"))

	homeDir := gethomedir(currentOptions.HomeDir)
	fatalIfFalse(homeDir != "", "home directory is empty")
	miniooniDir := path.Join(homeDir, ".miniooni")
	err := os.MkdirAll(miniooniDir, 0700)
	fatalOnError(err, "cannot create $HOME/.miniooni directory")

	// We cleanup the assets files used by versions of ooniprobe
	// older than v3.9.0, where we started embedding the assets
	// into the binary and use that directly. This cleanup doesn't
	// remove the whole directory but only known files inside it
	// and then the directory itself, if empty. We explicitly discard
	// the return value as it does not matter to us here.
	assetsDir := path.Join(miniooniDir, "assets")
	_, _ = assetsdir.Cleanup(assetsDir)

	log.Debugf("miniooni state directory: %s", miniooniDir)

	consentFile := path.Join(miniooniDir, "informed")
	fatalOnError(maybeWriteConsentFile(currentOptions.Yes, consentFile),
		"cannot write informed consent file")
	fatalIfFalse(canOpen(consentFile), riskOfRunningOONI)
	log.Info("miniooni home directory: $HOME/.miniooni")

	var proxyURL *url.URL
	if currentOptions.Proxy != "" {
		proxyURL = mustParseURL(currentOptions.Proxy)
	}

	kvstore2dir := filepath.Join(miniooniDir, "kvstore2")
	kvstore, err := kvstore.NewFS(kvstore2dir)
	fatalOnError(err, "cannot create kvstore2 directory")

	tunnelDir := filepath.Join(miniooniDir, "tunnel")
	err = os.MkdirAll(tunnelDir, 0700)
	fatalOnError(err, "cannot create tunnelDir")

	config := engine.SessionConfig{
		KVStore:         kvstore,
		Logger:          logger,
		ProxyURL:        proxyURL,
		SoftwareName:    softwareName,
		SoftwareVersion: softwareVersion,
		TorArgs:         currentOptions.TorArgs,
		TorBinary:       currentOptions.TorBinary,
		TunnelDir:       tunnelDir,
	}
	if currentOptions.ProbeServicesURL != "" {
		config.AvailableProbeServices = []model.OOAPIService{{
			Address: currentOptions.ProbeServicesURL,
			Type:    "https",
		}}
	}

	sess, err := engine.NewSession(ctx, config)
	fatalOnError(err, "cannot create measurement session")
	defer func() {
		sess.Close()
		log.Infof("whole session: recv %s, sent %s",
			humanize.SI(sess.KibiBytesReceived()*1024, "byte"),
			humanize.SI(sess.KibiBytesSent()*1024, "byte"),
		)
	}()
	log.Debugf("miniooni temporary directory: %s", sess.TempDir())

	log.Info("Looking up OONI backends; please be patient...")
	err = sess.MaybeLookupBackends()
	fatalOnError(err, "cannot lookup OONI backends")
	log.Info("Looking up your location; please be patient...")
	err = sess.MaybeLookupLocation()
	fatalOnError(err, "cannot lookup your location")
	log.Debugf("- IP: %s", sess.ProbeIP())
	log.Infof("- country: %s", sess.ProbeCC())
	log.Infof("- network: %s (%s)", sess.ProbeNetworkName(), sess.ProbeASNString())
	log.Infof("- resolver's IP: %s", sess.ResolverIP())
	log.Infof("- resolver's network: %s (%s)", sess.ResolverNetworkName(),
		sess.ResolverASNString())

	// We handle the oonirun experiment name specially. The user must specify
	// `miniooni -i {OONIRunURL} oonirun`` to run a OONI Run URL (v1 or v2).
	if experimentName == "oonirun" {
		ooniRunMain(ctx, sess, currentOptions, annotations)
		return
	}

	// Otherwise just run OONI experiments as we normally do.
	desc := &oonirun.Experiment{
		Annotations:    annotations,
		ExtraOptions:   extraOptions,
		Inputs:         currentOptions.Inputs,
		InputFilePaths: currentOptions.InputFilePaths,
		MaxRuntime:     currentOptions.MaxRuntime,
		Name:           experimentName,
		NoCollector:    currentOptions.NoCollector,
		NoJSON:         currentOptions.NoJSON,
		Random:         currentOptions.Random,
		ReportFile:     currentOptions.ReportFile,
		Session:        sess,
	}
	err = desc.Run(ctx)
	fatalOnError(err, "cannot run experiment")
}

// ooniRunMain runs the experiments described by the given OONI Run URLs. This
// function works with both v1 and v2 OONI Run URLs.
func ooniRunMain(ctx context.Context,
	sess *engine.Session, currentOptions Options, annotations map[string]string) {
	fatalIfTrue(
		len(currentOptions.Inputs) <= 0,
		"in oonirun mode you need to specify at least one URL using `-i URL`",
	)
	fatalIfTrue(
		len(currentOptions.InputFilePaths) > 0,
		"in oonirun mode you cannot specify any `-f FILE` file",
	)
	logger := sess.Logger()
	for _, URL := range currentOptions.Inputs {
		cfg := &oonirun.Config{
			AcceptChanges: currentOptions.Yes,
			Annotations:   annotations,
			KVStore:       sess.KeyValueStore(),
			MaxRuntime:    currentOptions.MaxRuntime,
			NoCollector:   currentOptions.NoCollector,
			NoJSON:        currentOptions.NoJSON,
			Random:        currentOptions.Random,
			ReportFile:    currentOptions.ReportFile,
			Session:       sess,
		}
		if err := oonirun.Measure(ctx, cfg, URL); err != nil {
			if errors.Is(err, oonirun.ErrNeedToAcceptChanges) {
				logger.Warnf("oonirun: to accept these changes, rerun adding `-y` to the command line")
				logger.Warnf("oonirun: we'll show this error every time the upstream link changes")
				panic("oonirun: need to accept changes using `-y`")
			}
			logger.Warnf("oonirun: Measure failed: %s", err.Error())
			continue
		}
	}
}
