package main

import (
	"net"

	"github.com/apex/log"
)

// IPTablesBlockingPolicies represents a set of iptable policies to blackhole
// endpoints using a specific docker bridge.
type IPTablesBlockingPolicies struct {
	config *Config
	dev    string
	shell  Shell
}

// NewIPTablesBlockingPolicies creates a new IPTables instance and applies
// the blackholing policies specified by config.
//
// Arguments:
//
// - config contains user-specified config;
//
// - dev is the network device to use (e.g., "eth0");
//
// - shell is the shell to use.
func NewIPTablesBlockingPolicies(
	config *Config, dev string, shell Shell) *IPTablesBlockingPolicies {
	ipt := &IPTablesBlockingPolicies{config: config, dev: dev, shell: shell}
	ipt.apply()
	return ipt
}

// apply applies iptables policies.
func (ipt *IPTablesBlockingPolicies) apply() {
	err := ipt.foreachBlackholedEndpoint(ipt.insertRule)
	FatalOnError(err, "cannot apply iptables policies to blackhole endpoints")
}

// Waive waives the policies.
func (ipt *IPTablesBlockingPolicies) Waive() {
	if err := ipt.foreachBlackholedEndpoint(ipt.deleteRule); err != nil {
		log.Warnf("cannot waive policies: %s", err.Error())
	}
}

type iptablesRuleMaker func(network, address, port string) []string

func (ipt *IPTablesBlockingPolicies) foreachBlackholedEndpoint(fn iptablesRuleMaker) error {
	for _, epnt := range ipt.config.Blackhole {
		address, port, err := net.SplitHostPort(epnt.Address)
		if err != nil {
			return err
		}
		cmd := NewCommandWithStdio("sudo", fn(epnt.Network, address, port)...)
		if err := ipt.shell.Run(cmd); err != nil {
			return err
		}
	}
	return nil
}

func (ipt *IPTablesBlockingPolicies) insertRule(network, address, port string) []string {
	return ipt.genericRule(network, address, port, "--insert")
}

func (ipt *IPTablesBlockingPolicies) deleteRule(network, address, port string) []string {
	return ipt.genericRule(network, address, port, "--delete")
}

func (ipt *IPTablesBlockingPolicies) genericRule(network, address, port, action string) []string {
	return []string{
		"iptables",
		action,
		"FORWARD",
		"--in-interface",
		ipt.dev,
		"--destination",
		address,
		"--protocol",
		network,
		"--destination-port",
		port,
		"--jump",
		"DROP",
	}
}
