// Command servertool is the local Signal-Server harness's helper (see signalserver/README.md):
// test certificates, the TLS test configuration, Maven's unpacking, and the TLS profile of a
// front end. It is research tooling, opt-in and outside CI, not part of probe-cli's measurements.
//
//	servertool certs OUT_DIR                     test certificates shaped like Signal's production chain
//	servertool config TEST_YML OUT_YML KSPASS    Signal-Server's test.yml with TLS front ends added
//	servertool unpack-maven ZIP DIR              Maven's distribution zip, its top directory as DIR
//	servertool tlsprofile TARGET...              a TLS front end's observable profile, as JSON
package main

import (
	"fmt"
	"os"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "servertool: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return usage()
	}
	rest := args[1:]
	switch {
	case args[0] == "certs" && len(rest) == 1:
		return writeCerts(rest[0])
	case args[0] == "config" && len(rest) == 3:
		return writeTestConfig(rest[0], rest[1], rest[2])
	case args[0] == "unpack-maven" && len(rest) == 2:
		return unpackMaven(rest[0], rest[1])
	case args[0] == "tlsprofile" && len(rest) > 0:
		return tlsProfiles(os.Stdout, rest)
	}
	return usage()
}

func usage() error {
	return fmt.Errorf("usage: servertool certs|config|unpack-maven|tlsprofile ... (see the package doc)")
}
