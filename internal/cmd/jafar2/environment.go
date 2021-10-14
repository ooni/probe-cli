package main

// Environment is the environment for jafar2.
type Environment struct {
	// DryRun disables destructive operations and only prints commands.
	DryRun bool

	// ForwardChain is the name of the iptables chain to use for custom filtering.
	ForwardChain string

	// NamespaceName is the name of the network namespace.
	NamespaceName string

	// NamespaceAddress is the address to assign to the namespace veth.
	NamespaceAddress string

	// NamespaceVeth is the veth to use in the namespace.
	NamespaceVeth string

	// LocalAddress is the address to assign to the local veth.
	LocalAddress string

	// LocalVeth is the name of the local veth.
	LocalVeth string

	// Netmask is the netmask to use.
	Netmask string
}

// NewEnvironment creates a new environment with default settings.
func NewEnvironment() *Environment {
	return &Environment{
		DryRun:           false,
		ForwardChain:     "JAFAR2_FORWARD",
		NamespaceName:    "jafar",
		NamespaceAddress: "10.14.17.11",
		NamespaceVeth:    "jafar1",
		LocalAddress:     "10.14.17.1",
		LocalVeth:        "jafar0",
		Netmask:          "24",
	}
}

// Validate ensures that the user-provided settings are ok.
func (env *Environment) Validate() error {
	if err := validateIPTablesChain(env.ForwardChain); err != nil {
		return err
	}
	if err := validateNetns(env.NamespaceName); err != nil {
		return err
	}
	if err := validateDevice(env.LocalVeth); err != nil {
		return err
	}
	if err := validateDevice(env.NamespaceVeth); err != nil {
		return err
	}
	if err := validateIP(env.LocalAddress); err != nil {
		return err
	}
	if err := validateIP(env.NamespaceAddress); err != nil {
		return err
	}
	if err := validateNetmask(env.Netmask); err != nil {
		return err
	}
	return nil
}
