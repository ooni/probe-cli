package main

import (
	"path/filepath"

	"github.com/pborman/getopt/v2"
)

// DestroyCmd implements the `jafar2 destroy` command.
type DestroyCmd struct{}

// Help returns the command help.
func (cmd *DestroyCmd) Help() string {
	return makeHelp(cmd, cmd.newGetoptParser(NewEnvironment()))
}

// BriefDescription returns a brief description of the command.
func (cmd *DestroyCmd) BriefDescription() string {
	return "destroys a previously-created network namespace"
}

// Main is the main of the `jafar2 destroy` command.
func (cmd *DestroyCmd) Main(args []string) {
	env := NewEnvironment()
	getopt := cmd.newGetoptParser(env)
	getopt.Parse(args)
	mustNotHavePositionalArguments(getopt, "destroy")
	fatalOnError(cmd.run(env, NewShell(env)))
}

// newGetoptParser returns the getopt parser for the create command.
func (cmd *DestroyCmd) newGetoptParser(env *Environment) *getopt.Set {
	getopt := getopt.New()
	getopt.SetProgram("jafar2 destroy")
	getopt.SetParameters("")
	getopt.FlagLong(&env.DryRun, "dry-run", 'n', "show what would have been done")
	getopt.FlagLong(&env.ForwardChain, "forward-chain", 0,
		"name of custom iptables forward chain")
	getopt.FlagLong(&env.NamespaceName, "namespace-name", 0,
		"name of the previously-created namespace")
	getopt.FlagLong(&env.LocalVeth, "local-veth", 0,
		"name assigned to the local veth")
	getopt.FlagLong(&env.Netmask, "netmask", 0,
		"netmask used when assigning IP addresses")
	return getopt
}

// run runs the destroy command.
func (cmd *DestroyCmd) run(env *Environment, sh Shell) error {
	if err := env.Validate(); err != nil {
		return err
	}
	device, err := sh.DefaultGatewayDevice()
	if err != nil {
		return err
	}
	// continue regardless of errors so we clean it up all
	cmd.destroyNamespace(sh, env)
	cmd.demasquerade(sh, env, device)
	cmd.destroyLocalVeth(sh, env)
	cmd.destroyFwdChain(sh, env)
	cmd.removeResolvConf(sh, env)
	return nil
}

// destroyNamespace destroys the network namespace.
func (cmd *DestroyCmd) destroyNamespace(sh Shell, env *Environment) error {
	return ShellRunf(sh, "ip netns del %s", env.NamespaceName)
}

// destroyLocalVeth destroys the local veth
func (cmd *DestroyCmd) destroyLocalVeth(sh Shell, env *Environment) error {
	return ShellRunf(sh, "ip link del %s", env.LocalVeth)
}

// demasquerade removes IP masquerading for the namespace.
func (cmd *DestroyCmd) demasquerade(sh Shell, env *Environment, device string) error {
	return ShellRunf(
		sh, "iptables -t nat -D POSTROUTING -s %s/%s -o %s -j MASQUERADE",
		env.LocalAddress, env.Netmask, device)
}

// destroyFwdChain destroys the custom forward chain.
func (cmd *DestroyCmd) destroyFwdChain(sh Shell, env *Environment) error {
	if err := ShellRunf(sh, "iptables -D FORWARD -j %s", env.ForwardChain); err != nil {
		return err
	}
	if err := ShellRunf(sh, "iptables -F %s", env.ForwardChain); err != nil {
		return err
	}
	return ShellRunf(sh, "iptables -X %s", env.ForwardChain)
}

// removeResolvConf removes the resolv.conf for the namespace.
func (cmd *DestroyCmd) removeResolvConf(sh Shell, env *Environment) error {
	dirpath := filepath.Join("/etc/netns", env.NamespaceName)
	return sh.RemoveAll(dirpath)
}
