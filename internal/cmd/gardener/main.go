// Command gardener helps with test-lists management.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/ooni/probe-cli/v3/internal/cmd/gardener/internal/dnsreport"
	"github.com/ooni/probe-cli/v3/internal/cmd/gardener/internal/sync"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/version"
	"github.com/spf13/cobra"
)

// repositoryDir is the path of the citizenlab/test-lists working copy.
const repositoryDir = "citizenlab-test-lists"

func main() {
	// select a colourful apex/log handler
	log.SetHandler(cli.New(os.Stderr))

	// create the root cobra command
	rootCmd := &cobra.Command{
		Use:     "gardener",
		Short:   "Gardener helps with test-lists management",
		Version: version.Version,
	}

	// create the sync subcommand
	syncCmd := &cobra.Command{
		Use:   "sync",
		Short: "Synchronizes the citizenlab/test-lists working copy",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			sc := &sync.Subcommand{
				RepositoryDir: repositoryDir,
			}
			sc.Main()
		},
	}
	rootCmd.AddCommand(syncCmd)

	// create the dnsreport subcommand
	dnsReportCmd := &cobra.Command{
		Use:   "dnsreport",
		Short: "Generates a DNS report from the citizenlab/test-lists working copy",
		Args:  cobra.NoArgs,
	}
	dnsReportCmdForce := dnsReportCmd.Flags().BoolP(
		"force",
		"f",
		false,
		"Force measuring again the test lists to regenerate the local cache",
	)
	dnsReportCmd.Run = func(cmd *cobra.Command, args []string) {
		sc := &dnsreport.Subcommand{
			CSVSummaryFile:        "dnsreport.csv",
			DNSOverHTTPSServerURL: "https://dns.google/dns-query",
			Force:                 *dnsReportCmdForce,
			JSONLCacheFile:        "dnsreport.jsonl",
			RepositoryDir:         repositoryDir,
		}
		runInterruptible(sc.Main)
	}
	rootCmd.AddCommand(dnsReportCmd)

	// execute the root command
	runtimex.Try0(rootCmd.Execute())
}

// mainSigCh is the signal where we post cancellation requests
var mainSigCh = make(chan os.Signal, 1024)

// runInterruptible runs the given function that takes in input a context
// and arrange for ^C to interrupt the function through the context.
func runInterruptible(fx func(ctc context.Context)) {
	// create cancellable context so we can interrupt fx
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// make sure we receive SIGINT
	signal.Notify(mainSigCh, syscall.SIGINT)

	// run fx in a background goroutine
	donech := make(chan any)
	go func() {
		defer close(donech)
		fx(ctx)
	}()

	select {
	case <-mainSigCh: // here we've been interrupted by a signal
		log.Warnf("interrupted by signal")

		// on signal interrupt the fx function
		cancel()

		log.Infof("waiting for background workers to terminate")

		// await for fx to terminate
		<-donech

	case <-donech: // this is the normal termination
	}
}
