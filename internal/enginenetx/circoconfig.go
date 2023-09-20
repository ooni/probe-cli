package enginenetx

// CircoConfig is the root of configuration for circumvention.
//
// (If convo is conversation, circo is circumvention.)
type CircoConfig struct {
	// Beacons contains configuration for beacons.
	//
	// From the point of view of OONI Probe a beacon is a host:
	//
	// 1. whose IP address we know in advance or we can dynamically discover using
	// currently not yet specified discovery mechanisms;
	//
	// 2. whose TLS stack is known to continue the handshake even if the incoming
	// SNI is wrong for the host, thus deferring TLS verification to the client (this
	// behavior is implicitly RECOMMENDED by RFC 6066 Sect. 3).
	Beacons map[string]CircoBeaconsDomain

	// Version is the version of the config.
	Version int
}

// beaconsIPAddrsForDomain returns either an empty list or a list of valid IP addresses
// for the given domain that could work as beacons.
func (c *CircoConfig) beaconsIPAddrsForDomain(domain string) (out []string) {
	// TODO(bassosimone): lock and unlock the configuration when it becomes dynamic
	if entry, good := c.Beacons[domain]; good {
		out = append(out, entry.IPAddrs...)
	}
	return
}

// allServerNamesForDomainIncludingDomain returns a list containing the domain itself as the first
// entry as well as zero or more additional SNIs for the given domain.
func (c *CircoConfig) allServerNamesForDomainIncludingDomain(domain string) (out []string) {
	// TODO(bassosimone): lock and unlock the configuration when it becomes dynamic
	out = append(out, domain)
	if entry, good := c.Beacons[domain]; good {
		out = append(out, entry.SNIs...)
	}
	return
}

// CircoBeaconsDomain contains beacon configuration for a domain.
type CircoBeaconsDomain struct {
	// IPAddrs lists the known IP addrs for the beacon.
	IPAddrs []string

	// SNIs lists possible SNIs for the beacon.
	SNIs []string
}

// CircoConfigVersion is the current version of the config used by circumvention.
const CircoConfigVersion = 0

// NewCircoConfig creates a new [*CircoConfig] using the default settings.
func NewCircoConfig() *CircoConfig {
	return &CircoConfig{
		Beacons: map[string]CircoBeaconsDomain{
			"api.ooni.io": {
				IPAddrs: []string{
					"162.55.247.208",
				},
				SNIs: []string{
					// TODO(bassosimone): we should pick the correct set of SNIs to use here
					"www.example.com",
					"www.example.org",
				},
			},
		},
		Version: CircoConfigVersion,
	}
}
