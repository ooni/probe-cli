package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/engine/legacy/assetsdir"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"github.com/ooni/probe-cli/v3/internal/humanize"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
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
	Limit            int64
	MaxRuntime       int64
	NoJSON           bool
	NoCollector      bool
	ProbeServicesURL string
	Proxy            string
	Random           bool
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
		&globalOptions.Limit, "limit", 0,
		"Limit the number of URLs tested by Web Connectivity", "N",
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
		&globalOptions.Yes, "yes", 0, "I accept the risk of running OONI",
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

func warnOnError(err error, msg string) {
	if err != nil {
		log.WithError(err).Warn(msg)
	}
}

func mustMakeMap(input []string) (output map[string]string) {
	output = make(map[string]string)
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

// limitRemoved is the text printed when the user uses --limit
const limitRemoved = `USAGE CHANGE: The --limit option has been removed in favor of
the --max-runtime option. Please, update your script to use --max-runtime
instead of --limit. The argument to --max-runtime is the maximum number
of seconds after which to stop running Web Connectivity.

This error message will be removed after 2021-11-01.
`

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
	fatalIfFalse(currentOptions.Limit == 0, limitRemoved)
	fatalIfTrue(currentOptions.Proxy != "" && currentOptions.Tunnel != "",
		tunnelAndProxy)
	if currentOptions.Tunnel != "" {
		currentOptions.Proxy = fmt.Sprintf("%s:///", currentOptions.Tunnel)
	}

	ctx := context.Background()

	extraOptions := mustMakeMap(currentOptions.ExtraOptions)
	annotations := mustMakeMap(currentOptions.Annotations)

	logger := &log.Logger{Level: log.InfoLevel, Handler: &logHandler{Writer: os.Stderr}}
	if currentOptions.Verbose {
		logger.Level = log.DebugLevel
	}
	if currentOptions.ReportFile == "" {
		currentOptions.ReportFile = "report.jsonl"
	}
	log.Log = logger

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
		config.AvailableProbeServices = []model.Service{{
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

	builder, err := sess.NewExperimentBuilder(experimentName)
	fatalOnError(err, "cannot create experiment builder")

	inputLoader := &engine.InputLoader{
		CheckInConfig: &model.CheckInConfig{
			RunType:  "manual",
			OnWiFi:   true, // meaning: not on 4G
			Charging: true,
		},
		InputPolicy:  builder.InputPolicy(),
		StaticInputs: currentOptions.Inputs,
		SourceFiles:  currentOptions.InputFilePaths,
		Session:      sess,
	}
	inputs, err := inputLoader.Load(context.Background())
	fatalOnError(err, "cannot load inputs")

	if currentOptions.Random {
		rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
		rnd.Shuffle(len(inputs), func(i, j int) {
			inputs[i], inputs[j] = inputs[j], inputs[i]
		})
	}

	err = builder.SetOptionsGuessType(extraOptions)
	fatalOnError(err, "cannot parse extraOptions")

	experiment := builder.NewExperiment()
	defer func() {
		log.Infof("experiment: recv %s, sent %s",
			humanize.SI(experiment.KibiBytesReceived()*1024, "byte"),
			humanize.SI(experiment.KibiBytesSent()*1024, "byte"),
		)
	}()

	submitter, err := engine.NewSubmitter(ctx, engine.SubmitterConfig{
		Enabled: !currentOptions.NoCollector,
		Session: sess,
		Logger:  log.Log,
	})
	fatalOnError(err, "cannot create submitter")

	saver, err := engine.NewSaver(engine.SaverConfig{
		Enabled:    !currentOptions.NoJSON,
		Experiment: experiment,
		FilePath:   currentOptions.ReportFile,
		Logger:     log.Log,
	})
	fatalOnError(err, "cannot create saver")

	inputProcessor := &engine.InputProcessor{
		Annotations: annotations,
		Experiment: &experimentWrapper{
			child: engine.NewInputProcessorExperimentWrapper(experiment),
			total: len(inputs),
		},
		Inputs:     inputs,
		MaxRuntime: time.Duration(currentOptions.MaxRuntime) * time.Second,
		Options:    currentOptions.ExtraOptions,
		Saver:      engine.NewInputProcessorSaverWrapper(saver),
		Submitter: submitterWrapper{
			child: engine.NewInputProcessorSubmitterWrapper(submitter),
		},
	}
	err = inputProcessor.Run(ctx)
	fatalOnError(err, "inputProcessor.Run failed")
}

type experimentWrapper struct {
	child engine.InputProcessorExperimentWrapper
	total int
}

func (ew *experimentWrapper) MeasureWithContext(
	ctx context.Context, idx int, input string) (*model.Measurement, error) {
	if input != "" {
		log.Infof("[%d/%d] running with input: %s", idx+1, ew.total, input)
	}
	measurement, err := ew.child.MeasureWithContext(ctx, idx, input)
	warnOnError(err, "measurement failed")
	// policy: we do not stop the loop if the measurement fails
	return measurement, nil
}

type submitterWrapper struct {
	child engine.InputProcessorSubmitterWrapper
}

func (sw submitterWrapper) Submit(ctx context.Context, idx int, m *model.Measurement) error {
	err := sw.child.Submit(ctx, idx, m)
	warnOnError(err, "submitting measurement failed")
	// policy: we do not stop the loop if measurement submission fails
	return nil
}
