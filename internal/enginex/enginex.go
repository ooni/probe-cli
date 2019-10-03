// Package enginex contains ooni/probe-engine extensions.
package enginex

import (
	"encoding/json"

	"github.com/apex/log"
	"github.com/ooni/probe-engine/model"
)

// Logger is the logger used by the engine.
var Logger = log.WithFields(log.Fields{
	"type": "engine",
})

// MakeGenericTestKeys casts the m.TestKeys to a map[string]interface{}.
//
// Ideally, all tests should have a clear Go structure, well defined, that
// will be stored in m.TestKeys as an interface. This is not already the
// case and it's just valid for tests written in Go. Until all tests will
// be written in Go, we'll keep this glue here to make sure we convert from
// the engine format to the cli format.
//
// This function will first attempt to cast directly to map[string]interface{},
// which is possible for MK tests, and then use JSON serialization and
// de-serialization only if that's required.
func MakeGenericTestKeys(m model.Measurement) (map[string]interface{}, error) {
	if result, ok := m.TestKeys.(map[string]interface{}); ok {
		return result, nil
	}
	data, err := json.Marshal(m.TestKeys)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	return result, err
}

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
