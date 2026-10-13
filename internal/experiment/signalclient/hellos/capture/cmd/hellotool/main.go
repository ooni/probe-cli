// Command hellotool turns packet captures of Signal's clients into the ClientHello files that
// signal_client embeds (hellos/*.bin), and checks them. It is research tooling for refreshing
// the captures (see hellos/README.md), not part of probe-cli's measurements.
//
//	hellotool extract PCAP SNI OUT.bin      the first ClientHello sent to SNI, normalized, with its JA4 and JA3
//	hellotool normalize FILE.bin...         zero the random values of existing ClientHello files
//	hellotool same A.bin B.bin...           check that ClientHello files differ only in SNI and padding
//	hellotool analyze PCAP [-records]       a capture's DNS queries, TCP connections and TLS records
//	hellotool check-okhttp VERSION CACHE [-write]   OkHttp still sets up Conscrypt sockets as verified
//	hellotool check-extract [-fingerprint PATH] [-keep DIR]   extract against real OpenSSL hellos
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
	records := flags.Bool("records", false, "analyze: also list each connection's TLS records")
	write := flags.Bool("write", false, "check-okhttp: record the current sources as verified")
	fingerprint := flags.String("fingerprint", "", "check-extract: also check this fingerprint.py (TCP and QUIC)")
	keep := flags.String("keep", "", "check-extract: keep each case's capture and -msg output under this directory")
	var rest []string // flags may come before, between or after the arguments
	for remaining := args[1:]; ; remaining = remaining[1:] {
		if err := flags.Parse(remaining); err != nil {
			return err
		}
		if remaining = flags.Args(); len(remaining) == 0 {
			break
		}
		rest = append(rest, remaining[0])
	}
	switch args[0] {
	case "extract":
		if len(rest) != 3 {
			return usage()
		}
		return extract(rest[0], rest[1], rest[2])
	case "normalize":
		return normalizeFiles(rest)
	case "same":
		return sameHello(rest)
	case "analyze":
		if len(rest) != 1 {
			return usage()
		}
		return analyze(rest[0], *records)
	case "check-okhttp":
		if len(rest) != 2 {
			return usage()
		}
		return checkOkHTTP(rest[0], rest[1], filepath.Join(captureDir(), "okhttp-conscrypt-tls.txt"), *write)
	case "check-extract":
		return checkExtract(*fingerprint, *keep)
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
