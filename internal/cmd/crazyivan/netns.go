package main

import (
	"path/filepath"

	"golang.org/x/sys/execabs"
)

// Netns is the network namespace we create for measurements.
type Netns struct {
	env *Environment
	sh  Shell
}

// NewNetns creates a new network namespace instance.
func NewNetns(env *Environment, sh Shell) *Netns {
	return &Netns{env: env, sh: sh}
}

// Create creates the network namespace.
func (nns *Netns) Create() error {
	device, err := nns.sh.DefaultGatewayDevice()
	if err != nil {
		return err
	}
	if err := nns.createNamespace(); err != nil {
		return err
	}
	if err := nns.createVethPair(); err != nil {
		return err
	}
	if err := nns.setLocalVethUp(); err != nil {
		return err
	}
	if err := nns.setNamespaceVethUp(); err != nil {
		return err
	}
	if err := nns.assignLocalAddress(); err != nil {
		return err
	}
	if err := nns.assignNamespaceAddress(); err != nil {
		return err
	}
	if err := nns.addDefaultRouteToNamespace(); err != nil {
		return err
	}
	if err := nns.masquerade(device); err != nil {
		return err
	}
	if err := nns.createFwdChain(); err != nil {
		return err
	}
	if err := nns.maybeBlockEndpoints(); err != nil {
		return err
	}
	return nns.writeResolvConf()
}

// createNamespace creates the network namespace.
func (nns *Netns) createNamespace() error {
	return ShellRunf(nns.sh, "ip netns add %s", nns.env.NamespaceName)
}

// createVethPair creates a pair of veth devices.
func (nns *Netns) createVethPair() error {
	return ShellRunf(nns.sh, "ip link add %s type veth peer netns %s name %s",
		nns.env.LocalVeth, nns.env.NamespaceName, nns.env.NamespaceVeth)
}

// setLocalVethUp brings up the local veth.
func (nns *Netns) setLocalVethUp() error {
	return ShellRunf(nns.sh, "ip link set dev %s up", nns.env.LocalVeth)
}

// setNamespaceVethUp brings up the remote veth.
func (nns *Netns) setNamespaceVethUp() error {
	return ShellRunf(nns.sh, "ip -n %s link set dev %s up",
		nns.env.NamespaceName, nns.env.NamespaceVeth)
}

// assignLocalAddress assigns the local address.
func (nns *Netns) assignLocalAddress() error {
	return ShellRunf(nns.sh, "ip address add %s/%s dev %s",
		nns.env.LocalAddress, nns.env.Netmask, nns.env.LocalVeth)
}

// assignNamespaceAddress assigns the remote address.
func (nns *Netns) assignNamespaceAddress() error {
	return ShellRunf(nns.sh, "ip -n %s address add %s/%s dev %s",
		nns.env.NamespaceName, nns.env.NamespaceAddress, nns.env.Netmask,
		nns.env.NamespaceVeth)
}

// addDefaultRouteToNamespace adds a default route inside the namespace.
func (nns *Netns) addDefaultRouteToNamespace() error {
	return ShellRunf(nns.sh, "ip -n %s route add default via %s dev %s",
		nns.env.NamespaceName, nns.env.LocalAddress, nns.env.NamespaceVeth)
}

// masquerade configures IP masquerading for the namespace.
func (nns *Netns) masquerade(device string) error {
	return ShellRunf(
		nns.sh, "iptables -t nat -A POSTROUTING -s %s/%s -o %s -j MASQUERADE",
		nns.env.LocalAddress, nns.env.Netmask, device)
}

// createFwdChain creates a custom iptables chain for filtering forwarded packets.
func (nns *Netns) createFwdChain() error {
	if err := ShellRunf(nns.sh, "iptables -N %s", nns.env.ForwardChain); err != nil {
		return err
	}
	return ShellRunf(nns.sh, "iptables -I FORWARD -j %s", nns.env.ForwardChain)
}

// maybeBlockEndpoints adds rules for block endpoints.
func (nns *Netns) maybeBlockEndpoints() error {
	for _, epnt := range nns.env.Block.Endpoints {
		err := nns.sh.Runv([]string{
			"iptables",
			"-t",
			"filter",
			"-A",
			nns.env.ForwardChain.String(),
			"-p",
			epnt.Network,
			"-d",
			epnt.Address,
			"--dport",
			epnt.Port,
			"-i",
			nns.env.LocalVeth.String(),
			"-j",
			"DROP",
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// writeResolvConf writes a resolv.conf for the namespace.
func (nns *Netns) writeResolvConf() error {
	dirpath := filepath.Join("/etc/netns", nns.env.NamespaceName.String())
	err := nns.sh.MkdirAll(dirpath, 0755)
	if err != nil {
		return err
	}
	filepath := filepath.Join(dirpath, "resolv.conf")
	data := []byte("nameserver 1.1.1.1\nnameserver 8.8.8.8\nnameserver 9.9.9.9\n")
	return nns.sh.WriteFile(filepath, data, 0644)
}

// Destroy destroys the network namespace.
func (nns *Netns) Destroy() error {
	device, err := nns.sh.DefaultGatewayDevice()
	if err != nil {
		return err
	}
	// continue regardless of errors so we clean it up all
	nns.destroyNamespace()
	nns.demasquerade(device)
	nns.destroyLocalVeth()
	nns.destroyFwdChain()
	nns.removeResolvConf()
	return nil
}

// destroyNamespace destroys the network namespace.
func (nns *Netns) destroyNamespace() error {
	return ShellRunf(nns.sh, "ip netns del %s", nns.env.NamespaceName)
}

// destroyLocalVeth destroys the local veth
func (nns *Netns) destroyLocalVeth() error {
	return ShellRunf(nns.sh, "ip link del %s", nns.env.LocalVeth)
}

// demasquerade removes IP masquerading for the namespace.
func (nns *Netns) demasquerade(device string) error {
	return ShellRunf(
		nns.sh, "iptables -t nat -D POSTROUTING -s %s/%s -o %s -j MASQUERADE",
		nns.env.LocalAddress, nns.env.Netmask, device)
}

// destroyFwdChain destroys the custom forward chain.
func (nns *Netns) destroyFwdChain() error {
	if err := ShellRunf(nns.sh, "iptables -D FORWARD -j %s", nns.env.ForwardChain); err != nil {
		return err
	}
	if err := ShellRunf(nns.sh, "iptables -F %s", nns.env.ForwardChain); err != nil {
		return err
	}
	return ShellRunf(nns.sh, "iptables -X %s", nns.env.ForwardChain)
}

// removeResolvConf removes the resolv.conf for the namespace.
func (nns *Netns) removeResolvConf() error {
	dirpath := filepath.Join("/etc/netns", nns.env.NamespaceName.String())
	return nns.sh.RemoveAll(dirpath)
}

// Execv executes the given argv inside the namespace.
func (nns *Netns) Execv(args []string) error {
	setpriv, err := execabs.LookPath("setpriv")
	if err != nil {
		return err
	}
	arguments := []string{
		"ip",
		"netns",
		"exec",
		nns.env.NamespaceName.String(),
		setpriv,
		"--clear-groups",
		"--inh-caps",
		"-all",
		"--ambient-caps",
		"-all",
		"--bounding-set",
		"-all",
		"--no-new-privs",
		"--reset-env",
	}
	if nns.env.Group != "" {
		arguments = append(arguments, "--regid")
		arguments = append(arguments, nns.env.Group.String())
	}
	if nns.env.User != "" {
		arguments = append(arguments, "--reuid")
		arguments = append(arguments, nns.env.Group.String())
	}
	arguments = append(arguments, "--")
	arguments = append(arguments, args...)
	return nns.sh.Runv(arguments)
}
