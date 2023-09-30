package dslnet

// Endpoint contains information to establish a TCP or QUIC connection and
// to initiate a measurement pipeline using such a connection.
type Endpoint struct {
	// Domain is the OPTIONAL domain from which we resolved IPAddress.
	Domain string

	// IPAddress is the MANDATORY IP address.
	IPAddress string

	// Network is the MANDATORY network ("tcp" or "udp").
	Network string

	// Port is the MANDATORY port.
	Port string

	// Tags contains OPTIONAL tags to tag OONI observations.
	Tags []string
}

// Clone clones the [Endpoint].
func (e Endpoint) Clone() Endpoint {
	return Endpoint{
		Domain:    e.Domain,
		IPAddress: e.IPAddress,
		Network:   e.Network,
		Port:      e.Port,
		Tags:      append([]string{}, e.Tags...),
	}
}
