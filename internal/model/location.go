package model

// LocationProvider is an interface that returns the current location. The
// [engine.Session] struct implements this interface.
type LocationProvider interface {
	ProbeASN() uint
	ProbeASNString() string
	ProbeCC() string
	ProbeIP() string
	ProbeNetworkName() string
	ResolverIP() string
}

// LocationASN contains ASN information related to a location.
type LocationASN struct {
	ASNumber     uint
	Organization string
}
