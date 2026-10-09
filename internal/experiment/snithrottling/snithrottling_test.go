package snithrottling

import (
	"context"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netemx"
)

func TestMeasurerNameAndVersion(t *testing.T) {
	m := NewExperimentMeasurer(Config{})
	if m.ExperimentName() != "sni_throttling" {
		t.Fatal("unexpected name")
	}
	if m.ExperimentVersion() != "0.1.0" {
		t.Fatal("unexpected version")
	}
}

func TestConfigDefaults(t *testing.T) {
	c := &Config{}
	if c.host() != "www.youtube.com" || c.fallbackHost() != "www.google.com" {
		t.Fatal("unexpected hosts")
	}
	if diff := cmp.Diff([]string{"www.youtube.com", "rr1---sn-4g5e6nzz.googlevideo.com"}, c.targetSNIs()); diff != "" {
		t.Fatal(diff)
	}
	if diff := cmp.Diff([]string{"www.google.com", "example.org"}, c.controlSNIs()); diff != "" {
		t.Fatal(diff)
	}
	if c.maxBytes() != 1<<20 || c.timeout() != 10*time.Second || c.rounds() != 1 {
		t.Fatal("unexpected limits")
	}
	c = &Config{TargetSNIs: "a  b", ControlSNIs: "c", MaxBytes: 7, Timeout: 3, Rounds: 2}
	if diff := cmp.Diff([]string{"a", "b"}, c.targetSNIs()); diff != "" {
		t.Fatal(diff)
	}
	if c.maxBytes() != 7 || c.timeout() != 3*time.Second || c.rounds() != 2 {
		t.Fatal("unexpected limits")
	}
}

func TestYoutubePlayerPath(t *testing.T) {
	body := []byte(`<script src="/s/player/5203c085/player_es6.vflset/en_US/base.js" nonce="x">`)
	if got := youtubePlayerPath(body); got != "/s/player/5203c085/player_es6.vflset/en_US/base.js" {
		t.Fatal("unexpected path", got)
	}
	if got := youtubePlayerPath([]byte("<html></html>")); got != "" {
		t.Fatal("unexpected path", got)
	}
}

// newDownload returns a download that completed the handshake.
func newDownload(role, sni string, status, length int64, goodput float64) *Download {
	return &Download{
		Role:         role,
		SNI:          sni,
		TLSHandshake: &model.ArchivalTLSOrQUICHandshakeResult{},
		StatusCode:   status,
		BodyLength:   length,
		GoodputKbps:  goodput,
	}
}

func TestAnalyze(t *testing.T) {
	failure := "generic_timeout_error"
	blockedHandshake := &Download{
		Role:         "target",
		SNI:          "blocked.com",
		TLSHandshake: &model.ArchivalTLSOrQUICHandshakeResult{Failure: &failure},
		Failure:      &failure,
	}
	tk := &TestKeys{
		Downloads: []*Download{
			newDownload("control", "control.com", 200, 1<<20, 10000),
			newDownload("control", "small.com", 200, 1000, 90000), // too small to count
			newDownload("target", "ok.com", 200, 1<<20, 8000),
			newDownload("target", "slow.com", 200, 200<<10, 500),
			blockedHandshake,
			newDownload("target", "stalled.com", 200, 0, 0),
			newDownload("target", "redirect.com", 302, 0, 0),
		},
		Verdicts: map[string]string{},
		Ratios:   map[string]float64{},
	}
	tk.analyze([]string{"ok.com", "slow.com", "blocked.com", "stalled.com", "redirect.com", "missing.com"})
	expect := map[string]string{
		"ok.com":       VerdictOK,
		"slow.com":     VerdictThrottled,
		"blocked.com":  VerdictBlocked,
		"stalled.com":  VerdictBlocked,
		"redirect.com": VerdictBlocked,
		"missing.com":  VerdictInconclusive,
	}
	if diff := cmp.Diff(expect, tk.Verdicts); diff != "" {
		t.Fatal(diff)
	}
	if tk.ControlGoodputKbps != 10000 {
		t.Fatal("unexpected control goodput", tk.ControlGoodputKbps)
	}
	sk := tk.MeasurementSummaryKeys().(*SummaryKeys)
	if diff := cmp.Diff([]string{"slow.com"}, sk.Throttled); diff != "" {
		t.Fatal(diff)
	}
	if diff := cmp.Diff([]string{"blocked.com", "redirect.com", "stalled.com"}, sk.Blocked); diff != "" {
		t.Fatal(diff)
	}
	if !sk.Anomaly() {
		t.Fatal("expected an anomaly")
	}
}

func TestAnalyzeWithoutControls(t *testing.T) {
	tk := &TestKeys{
		Downloads: []*Download{newDownload("target", "ok.com", 200, 1<<20, 8000)},
		Verdicts:  map[string]string{},
		Ratios:    map[string]float64{},
	}
	tk.analyze([]string{"ok.com"})
	if tk.Verdicts["ok.com"] != VerdictInconclusive {
		t.Fatal("unexpected verdict", tk.Verdicts["ok.com"])
	}
	if tk.MeasurementSummaryKeys().Anomaly() {
		t.Fatal("did not expect an anomaly")
	}
}

// runWithNetem runs the experiment against largefile.com in a netem scenario.
func runWithNetem(t *testing.T, configure func(env *netemx.QAEnv)) *TestKeys {
	env := netemx.MustNewScenario(netemx.InternetScenario)
	defer env.Close()
	if configure != nil {
		configure(env)
	}
	var tk *TestKeys
	env.Do(func() {
		m := &Measurer{
			Config: Config{
				Host:        "largefile.com",
				Path:        "/",
				TargetSNIs:  "largefile.com",
				ControlSNIs: "www.largefile.com example.org",
				MaxBytes:    1 << 20,
				Timeout:     5,
			},
			shuffle: func(int, func(int, int)) {}, // keep a predictable order
		}
		meas := &model.Measurement{MeasurementStartTimeSaved: time.Now()}
		args := &model.ExperimentArgs{
			Callbacks:   model.NewPrinterCallbacks(log.Log),
			Measurement: meas,
			Session:     &mocks.Session{MockLogger: func() model.Logger { return log.Log }},
		}
		if err := m.Run(context.Background(), args); err != nil {
			t.Fatal(err)
		}
		tk = meas.TestKeys.(*TestKeys)
	})
	return tk
}

func TestRunWithNetem(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}

	t.Run("without censorship", func(t *testing.T) {
		tk := runWithNetem(t, nil)
		if tk.Failure != nil {
			t.Fatal("unexpected failure", *tk.Failure)
		}
		if tk.Address != netemx.AddressLargeFileCom1+":443" {
			t.Fatal("unexpected address", tk.Address)
		}
		if len(tk.Downloads) != 3 {
			t.Fatal("unexpected number of downloads", len(tk.Downloads))
		}
		for _, d := range tk.Downloads {
			if d.Failure != nil {
				t.Fatal("unexpected failure", d.SNI, *d.Failure)
			}
			if d.BodyLength != 1<<20 {
				t.Fatal("unexpected body length", d.SNI, d.StatusCode, d.BodyLength)
			}
			if len(d.NetworkEvents) <= 0 {
				t.Fatal("expected network events")
			}
		}
		if tk.Verdicts["largefile.com"] != VerdictOK {
			t.Fatal("unexpected verdict", tk.Verdicts)
		}
	})

	t.Run("with the target SNI throttled", func(t *testing.T) {
		tk := runWithNetem(t, func(env *netemx.QAEnv) {
			env.DPIEngine().AddRule(&netem.DPIThrottleTrafficForTLSSNI{
				Delay:  300 * time.Millisecond,
				Logger: log.Log,
				PLR:    0.1,
				SNI:    "largefile.com",
			})
		})
		if tk.Verdicts["largefile.com"] != VerdictThrottled {
			t.Fatal("unexpected verdict", tk.Verdicts, tk.Ratios)
		}
	})

	t.Run("with the target SNI blocked", func(t *testing.T) {
		tk := runWithNetem(t, func(env *netemx.QAEnv) {
			env.DPIEngine().AddRule(&netem.DPIDropTrafficForTLSSNI{
				Logger: log.Log,
				SNI:    "largefile.com",
			})
		})
		if tk.Verdicts["largefile.com"] != VerdictBlocked {
			t.Fatal("unexpected verdict", tk.Verdicts)
		}
	})
}
