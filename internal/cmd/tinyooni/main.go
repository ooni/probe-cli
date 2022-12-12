package main

import (
	"os"

	"github.com/ooni/probe-cli/v3/internal/version"
	"github.com/spf13/cobra"
)

// GlobalOptions contains the global options.
type GlobalOptions struct {
	Emoji               bool
	HomeDir             string
	NoJSON              bool
	NoCollector         bool
	ProbeServicesURL    string
	Proxy               string
	RepeatEvery         int64
	ReportFile          string
	SnowflakeRendezvous string
	TorArgs             []string
	TorBinary           string
	Tunnel              string
	Verbose             bool
	Yes                 bool
}

func main() {
	var globalOptions GlobalOptions
	rootCmd := &cobra.Command{
		Use:     "tinyooni",
		Short:   "tinyooni is like miniooni but more experimental",
		Args:    cobra.NoArgs,
		Version: version.Version,
	}
	rootCmd.SetVersionTemplate("{{ .Version }}\n")
	flags := rootCmd.PersistentFlags()

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

	flags.StringVar(
		&globalOptions.SnowflakeRendezvous,
		"snowflake-rendezvous",
		"domain_fronting",
		"rendezvous method for --tunnel=torsf (one of: \"domain_fronting\" and \"amp\")",
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
		"tunnel to use to communicate with the OONI backend (one of: psiphon, tor, torsf)",
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

	registerWebConnectivity(rootCmd, &globalOptions)
	registerTelegram(rootCmd, &globalOptions)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
