package mocks

import "github.com/ooni/probe-cli/v3/internal/model"

type LocationProvider struct {
	MockProbeASN         func() uint
	MockProbeASNString   func() string
	MockProbeCC          func() string
	MockProbeIP          func() string
	MockProbeNetworkName func() string
	MockResolverIP       func() string
}

var _ model.LocationProvider = &LocationProvider{}

// ProbeASN calls MockProbeASN
func (loc *LocationProvider) ProbeASN() uint {
	return loc.MockProbeASN()
}

// ProbeASNString calls MockProbeASNString
func (loc *LocationProvider) ProbeASNString() string {
	return loc.MockProbeASNString()
}

// ProbeCC call MockProbeCC
func (loc *LocationProvider) ProbeCC() string {
	return loc.MockProbeCC()
}

// ProbeIP calls MockProbeIP
func (loc *LocationProvider) ProbeIP() string {
	return loc.MockProbeIP()
}

// ProbeNetworkName calls MockProbeNetworkName
func (loc *LocationProvider) ProbeNetworkName() string {
	return loc.MockProbeNetworkName()
}

// ResolverIP calls MockResolverIP
func (loc *LocationProvider) ResolverIP() string {
	return loc.MockResolverIP()
}
