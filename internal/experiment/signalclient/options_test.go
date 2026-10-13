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
	for _, config := range []Config{{Controls: "x"}, {Circumvention: "x"}, {Fronts: "x"}, {StaticAddresses: "x"}, {SignalCA: "x"}} {
		meas := &model.Measurement{ProbeCC: "IT", MeasurementStartTimeSaved: time.Now()}
		args := &model.ExperimentArgs{
			Callbacks:   model.NewPrinterCallbacks(model.DiscardLogger),
			Measurement: meas,
			Session:     &mocks.Session{MockLogger: func() model.Logger { return model.DiscardLogger }},
		}
		if err := NewExperimentMeasurer(config).Run(ctx, args); err == nil {
			t.Errorf("%+v accepted", config)
		}
		if tk := meas.TestKeys.(*TestKeys); tk.Targets != nil || tk.Queries != nil {
			t.Errorf("%+v: measured before failing: %+v", config, tk)
		}
	}
}

func TestParseOptionsCountry(t *testing.T) {
	m := NewExperimentMeasurer(Config{CircumventionCountry: " pk "}).(*Measurer)
	if opts, err := m.parseOptions("IT"); err != nil || opts.country != "PK" {
		t.Errorf("%+v %v", opts, err)
	}
	m = NewExperimentMeasurer(Config{}).(*Measurer)
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
