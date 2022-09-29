// Command miniooni is a simple binary for research and QA purposes
// with a CLI interface similar to MK and OONI Probe v2.x.
package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"runtime/debug"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/humanize"
	"github.com/ooni/probe-cli/v3/internal/legacy/assetsdir"
	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/registry"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/version"
	"github.com/spf13/cobra"
)

// Options contains the options you can set from the CLI.
type Options struct {
	Annotations      []string
	Emoji            bool
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
	Yes              bool
}

// main is the main function of miniooni.
func main() {
	var globalOptions Options
	rootCmd := &cobra.Command{
		Use:     "miniooni",
		Short:   "miniooni is OONI's research client",
		Args:    cobra.NoArgs,
		Version: version.Version,
	}
	rootCmd.SetVersionTemplate("{{ .Version }}\n")
	flags := rootCmd.PersistentFlags()

	flags.StringSliceVarP(
		&globalOptions.Annotations,
		"annotation",
		"A",
		[]string{},
		"add KEY=VALUE annotation to the report (can be repeated multiple times)",
	)

	flags.BoolVar(
		&globalOptions.Emoji,
		"emoji",
		false,
		"whether to use emojis when logging",
	)

	flags.StringVar(
		&globalOptions.HomeDir,
		"home",
		"",
		"force specific home directory",
	)

	flags.BoolVarP(
		&globalOptions.NoJSON,
		"no-json",
		"N",
		false,
		"disable writing to disk",
	)

	flags.BoolVarP(
		&globalOptions.NoCollector,
		"no-collector",
		"n",
		false,
		"do not submit measurements to the OONI collector",
	)

	flags.StringVar(
		&globalOptions.ProbeServicesURL,
		"probe-services",
		"",
		"URL of the OONI backend instance you want to use",
	)

	flags.StringVar(
		&globalOptions.Proxy,
		"proxy",
		"",
		"set proxy URL to communicate with the OONI backend (mutually exclusive with --tunnel)",
	)

	flags.Int64Var(
		&globalOptions.RepeatEvery,
		"repeat-every",
		0,
		"wait the given number of seconds and then repeat the same measurement",
	)

	flags.StringVarP(
		&globalOptions.ReportFile,
		"reportfile",
		"o",
		"",
		"set the output report file path (default: \"report.jsonl\")",
	)

	flags.StringSliceVar(
		&globalOptions.TorArgs,
		"tor-args",
		[]string{},
		"extra arguments for the tor binary (may be specified multiple times)",
	)

	flags.StringVar(
		&globalOptions.TorBinary,
		"tor-binary",
		"",
		"execute a specific tor binary",
	)

	flags.StringVar(
		&globalOptions.Tunnel,
		"tunnel",
		"",
		"tunnel to use to communicate with the OONI backend (one of: tor, psiphon)",
	)

	flags.BoolVarP(
		&globalOptions.Verbose,
		"verbose",
		"v",
		false,
		"increase verbosity level",
	)

	flags.BoolVarP(
		&globalOptions.Yes,
		"yes",
		"y",
		false,
		"assume yes as the answer to all questions",
	)

	rootCmd.MarkFlagsMutuallyExclusive("proxy", "tunnel")

	registerAllExperiments(rootCmd, &globalOptions)
	registerOONIRun(rootCmd, &globalOptions)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// TODO(bassosimone): the current implementation is basically a cobra application
// where we hammered the previous miniooni code to make it work. We should
// obviously strive for more correctness. For example, it's a bit disgusting
// that MainWithConfiguration is invoked for both oonirun and random experiments.

// registerOONIRun registers the oonirun subcommand
func registerOONIRun(rootCmd *cobra.Command, globalOptions *Options) {
	subCmd := &cobra.Command{
		Use:   "oonirun",
		Short: "Runs a given OONI Run v2 link",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			MainWithConfiguration(cmd.Use, globalOptions)
		},
	}
	rootCmd.AddCommand(subCmd)
	flags := subCmd.Flags()
	flags.StringSliceVarP(
		&globalOptions.Inputs,
		"input",
		"i",
		[]string{},
		"URL of the OONI Run v2 descriptor to run (may be specified multiple times)",
	)
	flags.StringSliceVarP(
		&globalOptions.InputFilePaths,
		"input-file",
		"f",
		[]string{},
		"Path to the OONI Run v2 descriptor to run (may be specified multiple times)",
	)
}

// registerAllExperiments registers a subcommand for each experiment
func registerAllExperiments(rootCmd *cobra.Command, globalOptions *Options) {
	for name, factory := range registry.AllExperiments {
		subCmd := &cobra.Command{
			Use:   name,
			Short: fmt.Sprintf("Runs the %s experiment", name),
			Args:  cobra.NoArgs,
			Run: func(cmd *cobra.Command, args []string) {
				MainWithConfiguration(cmd.Use, globalOptions)
			},
		}
		rootCmd.AddCommand(subCmd)
		flags := subCmd.Flags()

		switch factory.InputPolicy() {
		case model.InputOrQueryBackend,
			model.InputStrictlyRequired,
			model.InputOptional,
			model.InputOrStaticDefault:

			flags.StringSliceVarP(
				&globalOptions.InputFilePaths,
				"input-file",
				"f",
				[]string{},
				"path to file to supply test dependent input (may be specified multiple times)",
			)

			flags.StringSliceVarP(
				&globalOptions.Inputs,
				"input",
				"i",
				[]string{},
				"add test-dependent input (may be specified multiple times)",
			)

			flags.Int64Var(
				&globalOptions.MaxRuntime,
				"max-runtime",
				0,
				"maximum runtime in seconds for the experiment (zero means infinite)",
			)

			flags.BoolVar(
				&globalOptions.Random,
				"random",
				false,
				"randomize the inputs list",
			)

		default:
			// nothing
		}

		if doc := documentationForOptions(name, factory); doc != "" {
			flags.StringSliceVarP(
				&globalOptions.ExtraOptions,
				"option",
				"O",
				[]string{},
				doc,
			)
		}
	}
}

// MainWithConfiguration is the miniooni main with a specific configuration
// represented by the experiment name and the current options.
//
// This function will panic in case of a fatal error. It is up to you that
// integrate this function to either handle the panic of ignore it.
func MainWithConfiguration(experimentName string, currentOptions *Options) {
	runtimex.PanicOnError(engine.CheckEmbeddedPsiphonConfig(), "Invalid embedded psiphon config")
	if currentOptions.Tunnel != "" {
		currentOptions.Proxy = fmt.Sprintf("%s:///", currentOptions.Tunnel)
	}

	logHandler := logx.NewHandlerWithDefaultSettings()
	logHandler.Emoji = currentOptions.Emoji
	logger := &log.Logger{Level: log.InfoLevel, Handler: logHandler}
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
func mainSingleIteration(logger model.Logger, experimentName string, currentOptions *Options) {

	// We allow the inner code to fail but we stop propagating the panic here
	// such that --repeat-every works as intended anyway
	if currentOptions.RepeatEvery > 0 {
		defer func() {
			if r := recover(); r != nil {
				log.Warnf("recovered from panic: %+v\n%s\n", r, debug.Stack())
			}
		}()
	}

	extraOptions := mustMakeMapStringAny(currentOptions.ExtraOptions)
	annotations := mustMakeMapStringString(currentOptions.Annotations)

	ctx := context.Background()

	//Mon Jan 2 15:04:05 -0700 MST 2006
	log.Infof("Current time: %s", time.Now().Format("2006-01-02 15:04:05 MST"))

	homeDir := gethomedir(currentOptions.HomeDir)
	runtimex.Assert(homeDir != "", "home directory is empty")
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

func documentationForOptions(name string, factory *registry.Factory) string {
	var sb strings.Builder
	options, err := factory.Options()
	if err != nil || len(options) < 1 {
		return ""
	}
	fmt.Fprint(&sb, "Pass KEY=VALUE options to the experiment. Available options:\n")
	for name, info := range options {
		if info.Doc == "" {
			continue
		}
		fmt.Fprintf(&sb, "\n")
		fmt.Fprintf(&sb, "  -O, --option %s=<%s>\n", name, info.Type)
		fmt.Fprintf(&sb, "      %s\n", info.Doc)
	}
	return sb.String()
}
