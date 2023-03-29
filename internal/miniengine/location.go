package miniengine

//
// Probe location
//

// Location is the probe location.
type Location struct {
	// ProbeASN is the probe AS number.
	ProbeASN int64 `json:"probe_asn"`

	// ProbeASNString is the probe AS number as a string.
	ProbeASNString string `json:"probe_asn_string"`

	// ProbeCC is the probe country code.
	ProbeCC string `json:"probe_cc"`

	// ProbeNetworkName is the probe network name.
	ProbeNetworkName string `json:"probe_network_name"`

	// IP is the probe IP.
	ProbeIP string `json:"probe_ip"`

	// ResolverASN is the resolver ASN.
	ResolverASN int64 `json:"resolver_asn"`

	// ResolverASNString is the resolver AS number as a string.
	ResolverASNString string `json:"resolver_asn_string"`

	// ResolverIP is the resolver IP.
	ResolverIP string `json:"resolver_ip"`

	// ResolverNetworkName is the resolver network name.
	ResolverNetworkName string `json:"resolver_network_name"`
}
