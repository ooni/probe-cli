package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"

	"github.com/pborman/getopt/v2"
)

var (
	// regexpChain identifies valid iptables chains
	regexpChain = regexp.MustCompile("^[A-Z0-9_]{1,16}$")

	// regexpDevice identifies valid device names
	regexpDevice = regexp.MustCompile("^[a-zA-z0-9]{1,16}$")

	// regexpNetns identifies valid netns names
	regexpNetns = regexp.MustCompile("^[a-zA-z0-9]{1,16}$")

	// regexpNetmask identifies valid netmasks
	regexpNetmask = regexp.MustCompile("^[1-9][0-9]?$")
)

// validateIPTablesChain checks whether an iptables chain name is valid.
func validateIPTablesChain(chain string) error {
	if !regexpChain.MatchString(chain) {
		return errors.New("invalid name for iptables chain")
	}
	return nil
}

// validateNetns checks whether a netns name is valid.
func validateNetns(netns string) error {
	if !regexpNetns.MatchString(netns) {
		return errors.New("invalid name for network namespace")
	}
	return nil
}

// validateDevice checks whether a device name is valid.
func validateDevice(dev string) error {
	if !regexpDevice.MatchString(dev) {
		return errors.New("invalid name for network device")
	}
	return nil
}

// validateIP checks whether an IP address string is valid.
func validateIP(ip string) error {
	if net.ParseIP(ip) == nil {
		return errors.New("invalid value for IP address")
	}
	return nil
}

// validateNetmask checks whether a netmask string is valid.
func validateNetmask(netmask string) error {
	if !regexpNetmask.MatchString(netmask) {
		return errors.New("invalid value for netmask")
	}
	return nil
}

// makeHelp creates the help string for a command given the
// command itself and its options parser.
func makeHelp(cmd Command, getopt *getopt.Set) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "usage: %s %s%s\n\n", getopt.Program(), getopt.UsageLine(), getopt.Parameters())
	fmt.Fprintf(&sb, "This command %s.\n\n", cmd.BriefDescription())
	fmt.Fprintf(&sb, "Available options:\n")
	getopt.PrintOptions(&sb)
	return sb.String()
}

// mustNotHavePositionalArguments fails with an error message
// if the parsed options contain any positional argument.
func mustNotHavePositionalArguments(getopt *getopt.Set, name string) {
	if len(getopt.Args()) > 0 {
		fmt.Fprintf(os.Stderr, "jafar2 %s: unexpected positional arguments after options.\n", name)
		fmt.Fprintf(os.Stderr, "Run `jafar2 help %s` for more help.\n", name)
		os.Exit(1)
	}
}

// fatalOnError calls os.Exit(1) in case err is an error.
func fatalOnError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %s\n", err.Error())
		os.Exit(1)
	}
}
