package engine

import (
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/enginelocate"
	"github.com/ooni/probe-cli/v3/internal/experiment/dnscheck"
	"github.com/ooni/probe-cli/v3/internal/experiment/example"
	"github.com/ooni/probe-cli/v3/internal/experiment/signal"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestExperimentHonoursSharingDefaults(t *testing.T) {
	measure := func(info *enginelocate.Results) *model.Measurement {
		sess := &Session{location: info, kvStore: &kvstore.Memory{}}
		builder, err := sess.NewExperimentBuilder("example")
		if err != nil {
			t.Fatal(err)
		}
		exp := builder.NewExperiment().(*experiment)
		return exp.newMeasurement(model.NewOOAPIURLInfoWithDefaultCategoryAndCountry(""))
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

func TestExperimentMeasurementSummaryKeysNotImplemented(t *testing.T) {
	t.Run("the .Anomaly method returns false", func(t *testing.T) {
		sk := &ExperimentMeasurementSummaryKeysNotImplemented{}
		if sk.Anomaly() != false {
			t.Fatal("expected false")
		}
	})
}

func TestExperimentMeasurementSummaryKeys(t *testing.T) {
	t.Run("when the TestKeys implement MeasurementSummaryKeysProvider", func(t *testing.T) {
		tk := &signal.TestKeys{}
		meas := &model.Measurement{TestKeys: tk}
		sk := MeasurementSummaryKeys(meas)
		if _, good := sk.(*signal.SummaryKeys); !good {
			t.Fatal("not the expected type")
		}
	})

	t.Run("otherwise", func(t *testing.T) {
		// note: example does not implement SummaryKeys
		tk := &example.TestKeys{}
		meas := &model.Measurement{TestKeys: tk}
		sk := MeasurementSummaryKeys(meas)
		if _, good := sk.(*ExperimentMeasurementSummaryKeysNotImplemented); !good {
			t.Fatal("not the expected type")
		}
	})
}

// This test ensures that (*experiment).newMeasurement is working as intended.
func TestExperimentNewMeasurement(t *testing.T) {
	// create a session for testing that does not use the network at all
	sess := newSessionForTestingNoLookups(t)

	// create a conventional time for starting the experiment
	t0 := time.Date(2024, 6, 27, 10, 33, 0, 0, time.UTC)

	// create the experiment
	exp := &experiment{
		byteCounter: bytecounter.New(),
		callbacks:   model.NewPrinterCallbacks(model.DiscardLogger),
		measurer:    &dnscheck.Measurer{},
		mrep: &experimentMutableReport{
			mu:     sync.Mutex{},
			report: nil,
		},
		session:       sess,
		testName:      "dnscheck",
		testStartTime: t0.Format(model.MeasurementDateFormat),
		testVersion:   "0.1.0",
	}

	// create the richer input target
	target := &dnscheck.Target{
		Config: &dnscheck.Config{
			DefaultAddrs: "8.8.8.8 2001:4860:4860::8888",
			HTTP3Enabled: true,
		},
		URL: "https://dns.google/dns-query",
	}

	// create measurement
	meas := exp.newMeasurement(target)

	// make sure the input is correctly serialized
	t.Run("Input", func(t *testing.T) {
		if meas.Input != "https://dns.google/dns-query" {
			t.Fatal("unexpected meas.Input")
		}
	})

	// make sure the options are correctly serialized
	t.Run("Options", func(t *testing.T) {
		expectOptions := []string{`DefaultAddrs=8.8.8.8 2001:4860:4860::8888`, `HTTP3Enabled=true`}
		if diff := cmp.Diff(expectOptions, meas.Options); diff != "" {
			t.Fatal(diff)
		}
	})

	// make sure we've got the expected annotation keys
	t.Run("Annotations", func(t *testing.T) {
		const (
			expected = 1 << iota
			got
		)
		m := map[string]int{
			"architecture":   expected,
			"engine_name":    expected,
			"engine_version": expected,
			"go_version":     expected,
			"platform":       expected,
			"vcs_modified":   expected,
			"vcs_revision":   expected,
			"vcs_time":       expected,
			"vcs_tool":       expected,
		}
		for key := range meas.Annotations {
			m[key] |= got
		}
		for key, value := range m {
			if value != expected|got {
				t.Fatal("expected", expected|got, "for", key, "got", value)
			}
		}
	})

	// TODO(bassosimone,DecFox): this is the correct place where to
	// add more tests regarding how we create measurements.
}
