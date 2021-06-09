// Package ptxclient is a pluggable transports client. This package
// is only meant for testing and is not production ready.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/ptx"
)

func main() {
	mode := flag.String("m", "snowflake", "one of snowflake and obfs4")
	verbose := flag.Bool("v", false, "enable verbose mode")
	flag.Parse()
	if *verbose {
		log.SetLevel(log.DebugLevel)
	}
	var dialer ptx.ContextDialer
	switch *mode {
	case "snowflake":
		dialer = &ptx.SnowflakeDialer{}
	case "obfs4":
		dialer = ptx.DefaultTestingOBFS4Bridge()
	default:
		fmt.Fprintf(os.Stderr, "unknown -mode: %s\n", *mode)
		os.Exit(1)
	}
	listener := &ptx.Listener{
		ContextDialer: dialer,
		Logger:        log.Log,
	}
	if err := listener.Start(); err != nil {
		log.WithError(err).Fatal("listener.Start failed")
	}
	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, os.Interrupt)
	<-sigch
	listener.Stop()
}
