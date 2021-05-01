// +build linux

package iptables

import (
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/engine/shellx"
)

type linuxShell struct{}

func (s *linuxShell) createChains() (err error) {
	defer func() {
		if recover() != nil {
			// JUST KNOW WE'VE BEEN HERE
		}
	}()
	err = shellx.Run("sudo", "iptables", "-N", "JAFAR_INPUT")
	runtimex.PanicOnError(err, "cannot create JAFAR_INPUT chain")
	err = shellx.Run("sudo", "iptables", "-N", "JAFAR_OUTPUT")
	runtimex.PanicOnError(err, "cannot create JAFAR_OUTPUT chain")
	err = shellx.Run("sudo", "iptables", "-t", "nat", "-N", "JAFAR_NAT_OUTPUT")
	runtimex.PanicOnError(err, "cannot create JAFAR_NAT_OUTPUT chain")
	err = shellx.Run("sudo", "iptables", "-I", "OUTPUT", "-j", "JAFAR_OUTPUT")
	runtimex.PanicOnError(err, "cannot insert jump to JAFAR_OUTPUT")
	err = shellx.Run("sudo", "iptables", "-I", "INPUT", "-j", "JAFAR_INPUT")
	runtimex.PanicOnError(err, "cannot insert jump to JAFAR_INPUT")
	err = shellx.Run("sudo", "iptables", "-t", "nat", "-I", "OUTPUT", "-j", "JAFAR_NAT_OUTPUT")
	runtimex.PanicOnError(err, "cannot insert jump to JAFAR_NAT_OUTPUT")
	return nil
}

func (s *linuxShell) dropIfDestinationEquals(ip string) error {
	return shellx.Run("sudo", "iptables", "-A", "JAFAR_OUTPUT", "-d", ip, "-j", "DROP")
}

func (s *linuxShell) rstIfDestinationEqualsAndIsTCP(ip string) error {
	return shellx.Run(
		"sudo", "iptables", "-A", "JAFAR_OUTPUT", "--proto", "tcp", "-d", ip,
		"-j", "REJECT", "--reject-with", "tcp-reset",
	)
}

func (s *linuxShell) dropIfContainsKeywordHex(keyword string) error {
	return shellx.Run(
		"sudo", "iptables", "-A", "JAFAR_OUTPUT", "-m", "string", "--algo", "kmp",
		"--hex-string", keyword, "-j", "DROP",
	)
}

func (s *linuxShell) dropIfContainsKeyword(keyword string) error {
	return shellx.Run(
		"sudo", "iptables", "-A", "JAFAR_OUTPUT", "-m", "string", "--algo", "kmp",
		"--string", keyword, "-j", "DROP",
	)
}

func (s *linuxShell) rstIfContainsKeywordHexAndIsTCP(keyword string) error {
	return shellx.Run(
		"sudo", "iptables", "-A", "JAFAR_OUTPUT", "-m", "string", "--proto", "tcp", "--algo",
		"kmp", "--hex-string", keyword, "-j", "REJECT", "--reject-with", "tcp-reset",
	)
}

func (s *linuxShell) rstIfContainsKeywordAndIsTCP(keyword string) error {
	return shellx.Run(
		"sudo", "iptables", "-A", "JAFAR_OUTPUT", "-m", "string", "--proto", "tcp", "--algo",
		"kmp", "--string", keyword, "-j", "REJECT", "--reject-with", "tcp-reset",
	)
}

func (s *linuxShell) hijackDNS(address string) error {
	// Hijack any DNS query, like the Vodafone station does when using the
	// secure network feature. Our transparent proxies will use DoT, in order
	// to bypass this restriction and avoid routing loop.
	return shellx.Run(
		"sudo", "iptables", "-t", "nat", "-A", "JAFAR_NAT_OUTPUT", "-p", "udp",
		"--dport", "53", "-j", "DNAT", "--to", address,
	)
}

func (s *linuxShell) hijackHTTPS(address string) error {
	// We need to whitelist root otherwise the traffic sent by Jafar
	// itself will match the rule and loop.
	return shellx.Run(
		"sudo", "iptables", "-t", "nat", "-A", "JAFAR_NAT_OUTPUT", "-p", "tcp",
		"--dport", "443", "-m", "owner", "!", "--uid-owner", "0",
		"-j", "DNAT", "--to", address,
	)
}

func (s *linuxShell) hijackHTTP(address string) error {
	// We need to whitelist root otherwise the traffic sent by Jafar
	// itself will match the rule and loop.
	return shellx.Run(
		"sudo", "iptables", "-t", "nat", "-A", "JAFAR_NAT_OUTPUT", "-p", "tcp",
		"--dport", "80", "-m", "owner", "!", "--uid-owner", "0",
		"-j", "DNAT", "--to", address,
	)
}

func (s *linuxShell) waive() error {
	shellx.RunQuiet("sudo", "iptables", "-D", "OUTPUT", "-j", "JAFAR_OUTPUT")
	shellx.RunQuiet("sudo", "iptables", "-D", "INPUT", "-j", "JAFAR_INPUT")
	shellx.RunQuiet("sudo", "iptables", "-t", "nat", "-D", "OUTPUT", "-j", "JAFAR_NAT_OUTPUT")
	shellx.RunQuiet("sudo", "iptables", "-F", "JAFAR_INPUT")
	shellx.RunQuiet("sudo", "iptables", "-X", "JAFAR_INPUT")
	shellx.RunQuiet("sudo", "iptables", "-F", "JAFAR_OUTPUT")
	shellx.RunQuiet("sudo", "iptables", "-X", "JAFAR_OUTPUT")
	shellx.RunQuiet("sudo", "iptables", "-t", "nat", "-F", "JAFAR_NAT_OUTPUT")
	shellx.RunQuiet("sudo", "iptables", "-t", "nat", "-X", "JAFAR_NAT_OUTPUT")
	return nil
}

func newShell() *linuxShell {
	return &linuxShell{}
}
