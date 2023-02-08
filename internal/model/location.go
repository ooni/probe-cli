package model

// LocationProvider is an interface that returns the current location.
type LocationProvider interface {
	// ProbeASN is the ASN associated with ProbeIP.
	ProbeASN() uint

	// ProbeASNString returns the probe ASN as the AS%d string.
	ProbeASNString() string

	// ProbeCC is the country code associated with ProbeIP.
	ProbeCC() string

	// ProbeIP is the probe IP address.
	ProbeIP() string

	// ProbeNetworkName is the name of the ProbeASN.
	ProbeNetworkName() string

	// ResolverIP is the IP of the resolver.
	ResolverIP() string
}
