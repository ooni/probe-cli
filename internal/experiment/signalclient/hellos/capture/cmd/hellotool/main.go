// Command hellotool turns packet captures of Signal's clients into the ClientHello files that
// signal_client embeds (hellos/*.bin), and checks them. It is research tooling for refreshing
// the captures (see hellos/README.md), not part of probe-cli's measurements.
//
//	hellotool extract PCAP SNI OUT.bin      the first ClientHello sent to SNI, normalized, with its JA4 and JA3
//	hellotool normalize FILE.bin...         zero the random values of existing ClientHello files
//	hellotool same A.bin B.bin...           check that ClientHello files differ only in SNI and padding
//	hellotool analyze [-records] PCAP       a capture's DNS queries, TCP connections and TLS records
//	hellotool check-okhttp [-write] VERSION CACHE   OkHttp still sets up Conscrypt sockets as verified
//	hellotool check-extract [-keep DIR]     extract against real OpenSSL hellos
//
// Captures are Ethernet frames over IPv4 or IPv6, in pcap or pcapng files, as pasta --pcap writes
// them. The per-connection random values of a ClientHello (client random, legacy session id, key
// share public keys) are zeroed, keeping their lengths, so a refresh changes a file only when the
// handshake changes; uTLS generates fresh values anyway.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "hellotool: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return usage()
	}
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	var records, write *bool
	var keep *string
	switch args[0] { // each subcommand's flags, before its arguments
	case "analyze":
		records = flags.Bool("records", false, "also list each connection's TLS records")
	case "check-okhttp":
		write = flags.Bool("write", false, "record the current sources as verified")
	case "check-extract":
		keep = flags.String("keep", "", "keep each case's capture and -msg output under this directory")
	}
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	rest := flags.Args()
	switch {
	case args[0] == "extract" && len(rest) == 3:
		return extract(rest[0], rest[1], rest[2])
	case args[0] == "normalize":
		return normalizeFiles(rest)
	case args[0] == "same":
		return sameHello(rest)
	case args[0] == "analyze" && len(rest) == 1:
		return analyze(os.Stdout, rest[0], *records)
	case args[0] == "check-okhttp" && len(rest) == 2:
		return checkOkHTTP(rest[0], rest[1], filepath.Join(captureDir(), "okhttp-conscrypt-tls.txt"), *write)
	case args[0] == "check-extract" && len(rest) == 0:
		return checkExtract(*keep)
	}
	return usage()
}

// captureDir is hellos/capture, where the scripts and okhttp-conscrypt-tls.txt live: CAPTURE_DIR,
// or the working directory.
func captureDir() string {
	if dir := os.Getenv("CAPTURE_DIR"); dir != "" {
		return dir
	}
	dir, _ := os.Getwd()
	return dir
}

func usage() error {
	return fmt.Errorf("usage: hellotool extract|normalize|same|analyze|check-okhttp|check-extract ... (see the package doc)")
}
