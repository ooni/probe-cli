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

	// ResolverASN is the resolver ASN.
	ResolverASN() uint

	// ResolverASNString is the resolver ASN as the AS%d string.
	ResolverASNString() string

	// ResolverNetworkName is the name of the ResolverASN.
	ResolverNetworkName() string
}

// Location describes the probe's location.
type Location struct {
	// ProbeASN is the ASN associated with ProbeIP.
	ProbeASN int64

	// ProbeASNString returns the probe ASN as the AS%d string.
	ProbeASNString string

	// ProbeCC is the country code associated with ProbeIP.
	ProbeCC string

	// ProbeIP is the probe IP address.
	ProbeIP string

	// ProbeNetworkName is the name of the ProbeASN.
	ProbeNetworkName string

	// ResolverIP is the IP of the resolver.
	ResolverIP string

	// ResolverASN is the resolver ASN.
	ResolverASN int64

	// ResolverASNString is the resolver ASN as the AS%d string.
	ResolverASNString string

	// ResolverNetworkName is the name of the ResolverASN.
	ResolverNetworkName string
}
