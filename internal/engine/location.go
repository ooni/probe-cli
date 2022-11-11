package engine

// LocationProvider is an interface that returns the current location. The
// github.com/ooni/probe-cli/v3/internal/engine/session.Session implements it.
type LocationProvider interface {
	ProbeASN() uint
	ProbeASNString() string
	ProbeCC() string
	ProbeIP() string
	ProbeNetworkName() string
	ResolverIP() string
}
