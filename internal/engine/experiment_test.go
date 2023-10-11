package engine

import (
	"testing"

	"github.com/ooni/probe-cli/v3/internal/enginelocate"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestExperimentHonoursSharingDefaults(t *testing.T) {
	measure := func(info *enginelocate.Results) *model.Measurement {
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
		locationInfo *enginelocate.Results
		expect       func(*model.Measurement) bool
	}
	allspecs := []spec{{
		name:         "probeIP",
		locationInfo: &enginelocate.Results{ProbeIP: "8.8.8.8"},
		expect: func(m *model.Measurement) bool {
			return m.ProbeIP == model.DefaultProbeIP
		},
	}, {
		name:         "probeASN",
		locationInfo: &enginelocate.Results{ASN: 30722},
		expect: func(m *model.Measurement) bool {
			return m.ProbeASN == "AS30722"
		},
	}, {
		name:         "probeCC",
		locationInfo: &enginelocate.Results{CountryCode: "IT"},
		expect: func(m *model.Measurement) bool {
			return m.ProbeCC == "IT"
		},
	}, {
		name:         "probeNetworkName",
		locationInfo: &enginelocate.Results{NetworkName: "Vodafone Italia"},
		expect: func(m *model.Measurement) bool {
			return m.ProbeNetworkName == "Vodafone Italia"
		},
	}, {
		name:         "resolverIP",
		locationInfo: &enginelocate.Results{ResolverIP: "9.9.9.9"},
		expect: func(m *model.Measurement) bool {
			return m.ResolverIP == "9.9.9.9"
		},
	}, {
		name:         "resolverASN",
		locationInfo: &enginelocate.Results{ResolverASN: 44},
		expect: func(m *model.Measurement) bool {
			return m.ResolverASN == "AS44"
		},
	}, {
		name:         "resolverNetworkName",
		locationInfo: &enginelocate.Results{ResolverNetworkName: "Google LLC"},
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
