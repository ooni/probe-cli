package signalclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestParseControls(t *testing.T) {
	for in, want := range map[string]string{"": "on_failure", "on_failure": "on_failure", "always": "always", "off": "off"} {
		if got, err := parseControls(in); err != nil || got != want {
			t.Errorf("%q: %q %v", in, got, err)
		}
	}
	if _, err := parseControls("sometimes"); err == nil {
		t.Error("accepted an invalid value")
	}
}

func TestParseCircumvention(t *testing.T) {
	for in, want := range map[string]string{
		"":                        "",
		"all":                     "reflector_f,reflector_g,android_g,android_f",
		"android_f, reflector_g":  "reflector_g,android_f",
		"reflector_f,reflector_f": "reflector_f",
	} {
		got, err := parseCircumvention(in)
		if err != nil || strings.Join(got, ",") != want {
			t.Errorf("%q: %v %v", in, got, err)
		}
	}
	if _, err := parseCircumvention("reflector_x"); err == nil {
		t.Error("accepted an unknown route")
	}
}

// A bad option fails the run before anything is measured. The context is already canceled, so a
// bad option that were accepted would not reach the network either.
func TestRunRejectsBadOptions(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, config := range []Config{{Controls: "x"}, {Circumvention: "x"}, {CircumventionEnabledIn: "x"}, {Fronts: "x"},
		{StaticAddresses: "x"}, {SignalCA: "x"}} {
		meas := &model.Measurement{ProbeCC: "IT", MeasurementStartTimeSaved: time.Now()}
		args := &model.ExperimentArgs{
			Callbacks:   model.NewPrinterCallbacks(model.DiscardLogger),
			Measurement: meas,
			Session:     &mocks.Session{MockLogger: func() model.Logger { return model.DiscardLogger }},
			Target:      &RicherInputTarget{Config: &config},
		}
		if err := NewExperimentMeasurer().Run(ctx, args); err == nil {
			t.Errorf("%+v accepted", config)
		}
		if tk := meas.TestKeys.(*TestKeys); tk.Targets != nil || tk.Queries != nil {
			t.Errorf("%+v: measured before failing: %+v", config, tk)
		}
	}
}

// CircumventionEnabledIn measures the routes only where the probe's country is listed; an unknown
// country is in no list.
func TestCircumventionEnabledIn(t *testing.T) {
	for _, c := range []struct {
		enabledIn, probeCC string
		routes             int
	}{
		{"", "IT", 4}, // no restriction
		{"ir, RU", "IR", 4},
		{"ir, RU", "ru", 4},
		{"ir, RU", "IT", 0},
		{"IR", "", 0},
		{"IR", "ZZ", 0},
		{" , ", "IT", 4}, // names no country: no restriction
	} {
		opts, err := newMeasurer(Config{Circumvention: "all", CircumventionEnabledIn: c.enabledIn}).parseOptions(c.probeCC)
		if err != nil || len(opts.routes) != c.routes {
			t.Errorf("%q in %q: %v routes, err %v; want %d", c.probeCC, c.enabledIn, opts, err, c.routes)
		}
	}
	for _, bad := range []string{"IRN", "I1", "IR;RU"} {
		if _, err := parseCountries(bad); err == nil {
			t.Errorf("accepted %q", bad)
		}
	}
}

func TestParseOptionsCountry(t *testing.T) {
	m := newMeasurer(Config{CircumventionCountry: " pk "})
	if opts, err := m.parseOptions("IT"); err != nil || opts.country != "PK" {
		t.Errorf("%+v %v", opts, err)
	}
	m = newMeasurer(Config{})
	if opts, err := m.parseOptions("ir"); err != nil || opts.country != "IR" || opts.controls != ControlsOnFailure {
		t.Errorf("%+v %v", opts, err)
	}
}

func TestConfigUserAgent(t *testing.T) {
	cases := []struct {
		config Config
		client Client
		want   string
	}{
		{Config{}, ClientAndroid, "ooniprobe-signal_client/" + testVersion + " (+https://ooni.org)"},
		{Config{}, ClientLibsignal, "ooniprobe-signal_client/" + testVersion + " (+https://ooni.org) libsignal/0.105.0"},
		{Config{AppUserAgent: true}, ClientLibsignal, "Signal-Android/8.31.1 Android/34 libsignal/0.105.0"},
		{Config{AppUserAgent: true}, ClientDesktop, "Signal-Desktop/8.33.0 Linux"},
		{Config{UserAgent: "x", AppUserAgent: true}, ClientDesktop, "x"},
	}
	for _, c := range cases {
		if got := c.config.userAgent(c.client); got != c.want {
			t.Errorf("%+v %+v: got %q, want %q", c.config, c.client, got, c.want)
		}
	}
}
