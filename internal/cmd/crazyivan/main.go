package main

import (
	"fmt"
	"os"

	"github.com/ooni/probe-cli/v3/internal/getoptx"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// Environment is the environment in which we run.
type Environment struct {
	// Block contains the list of endpoints to block.
	Block OptEndpointsList

	// Cleanup runs a cleanup before setting up the namespace.
	Cleanup bool

	// DryRun disables destructive operations and only prints commands.
	DryRun bool

	// ForwardChain is the name of the iptables chain to use for custom filtering.
	ForwardChain OptIptablesChain

	// Group is the group to run as.
	Group OptUserGroup

	// Help indicates we want the user to see the help message.
	Help bool

	// NamespaceName is the name of the network namespace.
	NamespaceName OptNetns

	// NamespaceAddress is the address to assign to the namespace veth.
	NamespaceAddress OptIPAddress

	// NamespaceVeth is the veth to use in the namespace.
	NamespaceVeth OptDevice

	// LocalAddress is the address to assign to the local veth.
	LocalAddress OptIPAddress

	// LocalVeth is the name of the local veth.
	LocalVeth OptDevice

	// Netmask is the netmask to use.
	Netmask OptNetmask

	// User specifies the user that you want to run as.
	User OptUserGroup
}

// NewEnvironment creates a new environment with default settings.
func NewEnvironment() *Environment {
	return &Environment{
		Block:            OptEndpointsList{},
		Cleanup:          false,
		DryRun:           false,
		ForwardChain:     "CI_FORWARD",
		Group:            "root",
		Help:             false,
		NamespaceName:    "cinetns",
		NamespaceAddress: "10.14.17.11",
		NamespaceVeth:    "civeth1",
		LocalAddress:     "10.14.17.1",
		LocalVeth:        "civeth0",
		Netmask:          "24",
		User:             "root",
	}
}

// DescribeOption implements getoptx.Program.Describe.
func (env *Environment) DescribeOption(opt string) string {
	switch opt {
	case "block":
		return "Registers endpoints to block (e.g., 8.8.8.8:443/tcp)"
	case "cleanup":
		return "Runs cleanup before creating namespace (useful if a previous run crashed)"
	case "dry-run":
		return "Shows what would have been done if -n was not specified"
	case "forward-chain":
		return "Name for the custom iptables forward chain when to censor"
	case "group":
		return "Name of the group to run <command> as"
	case "help":
		return "Shows this help message"
	case "namespace-name":
		return "Name of the network namespace to create"
	case "namespace-address":
		return "Address to assign to the new namespace's virtual ethernet"
	case "namespace-veth":
		return "Name for the new namespace's virtual ethernet"
	case "local-address":
		return "Address to assign to the local virtual ethernet"
	case "local-veth":
		return "Name for the local virtual ethernet connected to the namespace's one"
	case "netmask":
		return "Netmask to use for addresses assigned to virtual ethernets"
	case "user":
		return "Name of the user to run <command> as"
	default:
		return ""
	}
}

// ProgramName implements getoptx.Program.ProgramName.
func (env *Environment) ProgramName() string {
	return "crazyivan"
}

// PositionalArguments implements getoptx.Program.PositionalArguments.
func (env *Environment) PositionalArguments() string {
	return "<command> [options]"
}

// ShortDescription implements getoptx.Program.ShortDescription
func (env *Environment) ShortDescription() string {
	return "Crazyivan runs a command inside a censored network namespace"
}

// ShortOptionName implements getoptx.Program.ShortOptionName
func (env *Environment) ShortOptionName(option string) rune {
	switch option {
	case "dry-run":
		return 'n'
	case "help":
		return 'h'
	default:
		return 0
	}
}

// AfterParsingChecks implements getoptx.Program.AfterParsingChecks.
func (env *Environment) AfterParsingChecks(getopt *getoptx.Parser) {
	if env.Help {
		getopt.PrintLongUsage(os.Stdout)
		os.Exit(0)
	}
	if len(getopt.PositionalArgs()) < 1 {
		fmt.Fprintf(os.Stderr, "%s: fatal: no command specified.\n", env.ProgramName())
		getopt.PrintShortUsage(os.Stderr)
		os.Exit(1)
	}
}

func main() {
	defer runtimex.TrapPanics()
	env := NewEnvironment()
	getopt := getoptx.NewParser(env)
	getopt.Parse(os.Args)
	arguments := getopt.PositionalArgs()
	sh := env.NewShell()
	netns := NewNetns(env, sh)
	if env.Cleanup {
		netns.Destroy() // start afresh
	}
	runtimex.PanicOnError(netns.Create(), "fatal")
	defer netns.Destroy()
	runtimex.PanicOnError(netns.Execv(arguments), "fatal")
}
