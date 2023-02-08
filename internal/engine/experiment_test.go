package engine

import (
	"testing"

	"github.com/ooni/probe-cli/v3/internal/geolocate"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestExperimentHonoursSharingDefaults(t *testing.T) {
	measure := func(info *geolocate.Results) *model.Measurement {
		sess := &Session{location: info}
		builder, err := sess.NewExperimentBuilder("example")
		if err != nil {
			t.Fatal(err)
		}
		exp := builder.NewExperiment().(*experiment)
		return exp.newMeasurement("")
	}
	type spec struct {
		name         string
		locationInfo *geolocate.Results
		expect       func(*model.Measurement) bool
	}
	allspecs := []spec{{
		name:         "probeIP",
		locationInfo: &geolocate.Results{IPAddr: "8.8.8.8"},
		expect: func(m *model.Measurement) bool {
			return m.ProbeIP == model.DefaultProbeIP
		},
	}, {
		name:         "probeASN",
		locationInfo: &geolocate.Results{ASN: 30722},
		expect: func(m *model.Measurement) bool {
			return m.ProbeASN == "AS30722"
		},
	}, {
		name:         "probeCC",
		locationInfo: &geolocate.Results{CountryCode: "IT"},
		expect: func(m *model.Measurement) bool {
			return m.ProbeCC == "IT"
		},
	}, {
		name:         "probeNetworkName",
		locationInfo: &geolocate.Results{NetworkName: "Vodafone Italia"},
		expect: func(m *model.Measurement) bool {
			return m.ProbeNetworkName == "Vodafone Italia"
		},
	}, {
		name:         "resolverIP",
		locationInfo: &geolocate.Results{ResolverIPAddr: "9.9.9.9"},
		expect: func(m *model.Measurement) bool {
			return m.ResolverIP == "9.9.9.9"
		},
	}, {
		name:         "resolverASN",
		locationInfo: &geolocate.Results{ResolverASNumber: 44},
		expect: func(m *model.Measurement) bool {
			return m.ResolverASN == "AS44"
		},
	}, {
		name:         "resolverNetworkName",
		locationInfo: &geolocate.Results{ResolverASNetworkName: "Google LLC"},
		expect: func(m *model.Measurement) bool {
			return m.ResolverNetworkName == "Google LLC"
		},
	}}
	for _, spec := range allspecs {
		t.Run(spec.name, func(t *testing.T) {
			if !spec.expect(measure(spec.locationInfo)) {
				t.Fatal("expectation failed")
			}
		})
	}
}
