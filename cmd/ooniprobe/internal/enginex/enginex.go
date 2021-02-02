// Package enginex contains ooni/probe-engine extensions.
package enginex

import (
	"github.com/apex/log"
)

// Logger is the logger used by the engine.
var Logger = log.WithFields(log.Fields{
	"type": "engine",
})

// LocationProvider is an interface that returns the current location. The
// github.com/ooni/probe-engine/session.Session implements it.
type LocationProvider interface {
	ProbeASN() uint
	ProbeASNString() string
	ProbeCC() string
	ProbeIP() string
	ProbeNetworkName() string
	ResolverIP() string
}
