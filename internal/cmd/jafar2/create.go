package main

import (
	"path/filepath"

	"github.com/pborman/getopt/v2"
)

// CreateCmd implements the `jafar2 create` command.
type CreateCmd struct{}

// Help returns the command help.
func (cmd *CreateCmd) Help() string {
	return makeHelp(cmd, cmd.newGetoptParser(NewEnvironment()))
}

// BriefDescription returns a brief description of the command.
func (cmd *CreateCmd) BriefDescription() string {
	return "creates the network namespace for running tests"
}

// Main is the main of the `jafar2 create` command.
func (cmd *CreateCmd) Main(args []string) {
	env := NewEnvironment()
	getopt := cmd.newGetoptParser(env)
	getopt.Parse(args)
	mustNotHavePositionalArguments(getopt, "create")
	fatalOnError(cmd.run(env, NewShell(env)))
}

// newGetoptParser returns the getopt parser for the create command.
func (cmd *CreateCmd) newGetoptParser(env *Environment) *getopt.Set {
	getopt := getopt.New()
	getopt.SetProgram("jafar2 create")
	getopt.SetParameters("")
	getopt.FlagLong(&env.DryRun, "dry-run", 'n', "show what would have been done")
	getopt.FlagLong(&env.ForwardChain, "forward-chain", 0,
		"name of custom iptables forward chain")
	getopt.FlagLong(&env.NamespaceName, "namespace-name", 0,
		"name for the new namespace")
	getopt.FlagLong(&env.NamespaceAddress, "namespace-address", 0,
		"IP address for the new namespace's veth")
	getopt.FlagLong(&env.NamespaceVeth, "namespace-veth", 0,
		"name for the new namespace's veth")
	getopt.FlagLong(&env.LocalAddress, "local-address", 0,
		"IP address for the local veth")
	getopt.FlagLong(&env.LocalVeth, "local-veth", 0,
		"name for the local veth")
	getopt.FlagLong(&env.Netmask, "netmask", 0,
		"netmask for IP addresses")
	return getopt
}

// run runs the create command.
func (cmd *CreateCmd) run(env *Environment, sh Shell) error {
	if err := env.Validate(); err != nil {
		return err
	}
	device, err := sh.DefaultGatewayDevice()
	if err != nil {
		return err
	}
	if err := cmd.createNamespace(sh, env); err != nil {
		return err
	}
	if err := cmd.createVethPair(sh, env); err != nil {
		return err
	}
	if err := cmd.setLocalVethUp(sh, env); err != nil {
		return err
	}
	if err := cmd.setNamespaceVethUp(sh, env); err != nil {
		return err
	}
	if err := cmd.assignLocalAddress(sh, env); err != nil {
		return err
	}
	if err := cmd.assignNamespaceAddress(sh, env); err != nil {
		return err
	}
	if err := cmd.addDefaultRouteToNamespace(sh, env); err != nil {
		return err
	}
	if err := cmd.masquerade(sh, env, device); err != nil {
		return err
	}
	if err := cmd.createFwdChain(sh, env); err != nil {
		return err
	}
	return cmd.writeResolvConf(sh, env)
}

// createNamespace creates the network namespace.
func (cmd *CreateCmd) createNamespace(sh Shell, env *Environment) error {
	return ShellRunf(sh, "ip netns add %s", env.NamespaceName)
}

// createVethPair creates a pair of veth devices.
func (cmd *CreateCmd) createVethPair(sh Shell, env *Environment) error {
	return ShellRunf(sh, "ip link add %s type veth peer netns jafar name %s",
		env.LocalVeth, env.NamespaceVeth)
}

// setLocalVethUp brings up the local veth.
func (cmd *CreateCmd) setLocalVethUp(sh Shell, env *Environment) error {
	return ShellRunf(sh, "ip link set dev %s up", env.LocalVeth)
}

// setNamespaceVethUp brings up the remote veth.
func (cmd *CreateCmd) setNamespaceVethUp(sh Shell, env *Environment) error {
	return ShellRunf(sh, "ip -n %s link set dev %s up",
		env.NamespaceName, env.NamespaceVeth)
}

// assignLocalAddress assigns the local address.
func (cmd *CreateCmd) assignLocalAddress(sh Shell, env *Environment) error {
	return ShellRunf(sh, "ip address add %s/%s dev %s",
		env.LocalAddress, env.Netmask, env.LocalVeth)
}

// assignNamespaceAddress assigns the remote address.
func (cmd *CreateCmd) assignNamespaceAddress(sh Shell, env *Environment) error {
	return ShellRunf(sh, "ip -n %s address add %s/%s dev %s",
		env.NamespaceName, env.NamespaceAddress, env.Netmask, env.NamespaceVeth)
}

// addDefaultRouteToNamespace adds a default route inside the namespace.
func (cmd *CreateCmd) addDefaultRouteToNamespace(sh Shell, env *Environment) error {
	return ShellRunf(sh, "ip -n %s route add default via %s dev %s",
		env.NamespaceName, env.LocalAddress, env.NamespaceVeth)
}

// masquerade configures IP masquerading for the namespace.
func (cmd *CreateCmd) masquerade(sh Shell, env *Environment, device string) error {
	return ShellRunf(
		sh, "iptables -t nat -A POSTROUTING -s %s/%s -o %s -j MASQUERADE",
		env.LocalAddress, env.Netmask, device)
}

// createFwdChain creates a custom iptables chain for filtering forwarded packets.
func (cmd *CreateCmd) createFwdChain(sh Shell, env *Environment) error {
	if err := ShellRunf(sh, "iptables -N %s", env.ForwardChain); err != nil {
		return err
	}
	return ShellRunf(sh, "iptables -I FORWARD -j %s", env.ForwardChain)
}

// writeResolvConf writes a resolv.conf for the namespace.
func (cmd *CreateCmd) writeResolvConf(sh Shell, env *Environment) error {
	dirpath := filepath.Join("/etc/netns", env.NamespaceName)
	err := sh.MkdirAll(dirpath, 0755)
	if err != nil {
		return err
	}
	filepath := filepath.Join(dirpath, "resolv.conf")
	data := []byte("nameserver 1.1.1.1\nnameserver 8.8.8.8\nnameserver 9.9.9.9\n")
	return sh.WriteFile(filepath, data, 0644)
}
