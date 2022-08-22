package model

//
// Definition of a VPN experiment.
//

// TODO refactor this into interfaces shared by openvpn and wg experiments.

type VPNExperiment struct {
	// Provider is the entity to which the endpoints belong. We might want
	// to keep a list of known providers (for which we have experiments).
	// If the provider is not known to OONI probe, it should be marked as
	// "unknown".
	Provider string
	// Hostname is the Hostname for the VPN Endpoint
	Hostname string
	// Port is the Port for the VPN Endpoint
	Port string
	// Protocol is the VPN protocol: openvpn, wg
	Protocol string
	// Transport is the underlying protocol: udp, tcp
	Transport string
	// Obfuscation is any obfuscation used for the tunnel: none, obfs4, ...
	Obfuscation string
	// Config is a pointer to a VPNConfig struct.
	Config *VPNConfig
}

type VPNConfig struct {
	Cipher   string
	Auth     string
	Compress string
	Ca       string
	Cert     string
	Key      string
}
