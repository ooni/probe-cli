package quicfp

import (
	"context"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

type nullLogger struct{}

func (nullLogger) Debug(msg string)                       {}
func (nullLogger) Debugf(format string, v ...interface{}) {}
func (nullLogger) Info(msg string)                        {}
func (nullLogger) Infof(format string, v ...interface{})  {}
func (nullLogger) Warn(msg string)                        {}
func (nullLogger) Warnf(format string, v ...interface{})  {}

type nullSession struct{ model.ExperimentSession }

func (nullSession) Logger() model.Logger { return nullLogger{} }

// runMeasurerAgainst runs the full quicfp vector suite against target and
// returns the resulting TestKeys, exactly like the CLI does.
func runMeasurerAgainst(t *testing.T, target string, repetitions int64, timeout time.Duration) *TestKeys {
	t.Helper()
	cfg := Config{
		Repetitions:   repetitions,
		VectorTimeout: int64(timeout / time.Millisecond),
	}
	m := &Measurer{}
	measurement := &model.Measurement{
		MeasurementStartTimeSaved: time.Now(),
	}
	args := &model.ExperimentArgs{
		Measurement: measurement,
		Session:     nullSession{},
		Target:      &Target{Config: &cfg, URL: target},
	}
	if err := m.Run(context.Background(), args); err != nil {
		t.Fatalf("Measurer.Run failed: %v", err)
	}
	tk, ok := measurement.TestKeys.(*TestKeys)
	if !ok {
		t.Fatal("TestKeys has unexpected type")
	}
	return tk
}

// TestIntegrationMockCensorNoFiltering is the control case for
// TestIntegrationMockCensorMatchesSignature: a mockCensor configured with
// an empty policy (Signature{Vectors: map[string]Outcome{}}) forwards
// every datagram it classifies, since decide() defaults to forward
// whenever a vector has no policy entry. This still exercises the full
// classification pipeline (every datagram is decrypted and inspected
// exactly as in the filtered case), just with a router that never chooses
// to drop anything.
//
// This is the baseline the entire differential design (RunDifferential,
// Control vs. Candidate) depends on: if a "clean" path can't reliably
// produce all-bypassed results, then a divergence observed against a real
// Candidate target would be meaningless -- it could just as easily be a
// bug making some vectors look blocked regardless of what's actually on
// the wire. This test proves that isn't happening: every single vector
// (all 43, matched or not by the classifier) comes back Bypassed with
// nothing else running interference.
func TestIntegrationMockCensorNoFiltering(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	backend, err := startMockBackend()
	if err != nil {
		t.Fatalf("starting mock backend: %v", err)
	}
	defer backend.close()

	noFiltering := Signature{Device: "no-filtering-control", Vectors: map[string]Outcome{}}
	censor, err := startMockCensor(noFiltering, backend.addr())
	if err != nil {
		t.Fatalf("starting mock censor: %v", err)
	}
	defer censor.close()

	tk := runMeasurerAgainst(t, censor.addr().String(), 3, 300*time.Millisecond)
	observed := OutcomeMap(tk)

	vectors, err := AllVectors("example.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(observed) != len(vectors) {
		t.Fatalf("expected an outcome for all %d vectors, got %d", len(vectors), len(observed))
	}

	var notBypassed []string
	for _, v := range vectors {
		if got := observed[v.Name]; got != OutcomeBypassed {
			notBypassed = append(notBypassed, v.Name+"="+got.String())
		}
	}
	if len(notBypassed) > 0 {
		t.Errorf("with no filtering applied, expected every vector to be bypassed; got non-bypassed results: %v", notBypassed)
		t.Logf("dropped anyway (should be empty): %v", censor.dropLog)
	}

	// The control also underpins the ranking side: with a genuinely empty
	// policy, the router drops nothing, so it should never appear to
	// resemble any real device's block pattern.
	if len(censor.dropLog) != 0 {
		t.Errorf("no-filtering censor should never drop anything, but dropped: %v", censor.dropLog)
	}
}

// TestIntegrationMockCensorMatchesSignature is the integration test: for
// every real device in table9_from_paper.json, it builds a software router
// (mockCensor) that enforces exactly that device's published block/bypass
// policy -- classifying each datagram with the same DPI logic validated in
// wireclienthello_classify_test.go -- runs quicfp's real Measurer against
// it over loopback, and checks two things end to end:
//  1. quicfp's own observed Outcome per vector matches the policy the
//     router was configured with, for every vector the policy actually has
//     an opinion on.
//  2. Feeding those observed outcomes into the real fingerprint-matching
//     engine (RankSignatures) correctly identifies which device the
//     router was impersonating, ranked first with a high score.
//
// This is the strongest test in the suite: it doesn't just check that
// packets are shaped correctly (quicfp_test.go's round-trip tests) or that
// the signature data is internally consistent (signatures_paper_test.go)
// -- it proves the whole pipeline (craft -> a real DPI-style decision ->
// response or silence -> aggregation -> identification) recovers a known
// ground truth, using a real UDP round trip rather than mocked function
// calls.
func TestIntegrationMockCensorMatchesSignature(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	allSigs, err := LoadSignatures("testdata/table9_from_paper.json")
	if err != nil {
		t.Fatal(err)
	}
	weights := DeriveWeights(allSigs)

	for _, policy := range allSigs {
		if isMetaSignature(policy) {
			continue
		}
		policy := policy
		t.Run(policy.Device, func(t *testing.T) {
			backend, err := startMockBackend()
			if err != nil {
				t.Fatalf("starting mock backend: %v", err)
			}
			defer backend.close()

			censor, err := startMockCensor(policy, backend.addr())
			if err != nil {
				t.Fatalf("starting mock censor: %v", err)
			}
			defer censor.close()

			tk := runMeasurerAgainst(t, censor.addr().String(), 3, 300*time.Millisecond)
			observed := OutcomeMap(tk)

			var mismatches int
			for name, wantOutcome := range policy.Vectors {
				if wantOutcome == OutcomeUnknown {
					continue // policy has no opinion; nothing to check
				}
				got, ok := observed[name]
				if !ok {
					t.Errorf("vector %q: no observed outcome at all", name)
					continue
				}
				if got != wantOutcome {
					mismatches++
					t.Errorf("vector %q: observed %s, policy says %s", name, got, wantOutcome)
				}
			}
			if mismatches > 0 {
				t.Logf("forward log (%d): %v", len(censor.forwardLog), censor.forwardLog)
				t.Logf("drop log (%d): %v", len(censor.dropLog), censor.dropLog)
			}

			ranked := RankSignatures(observed, allSigs, weights)
			if len(ranked) == 0 {
				t.Fatal("no ranked results")
			}
			top := ranked[0]
			if top.Device != policy.Device {
				t.Errorf("fingerprint ranking identified %q (score=%.2f) instead of the actual injected policy %q",
					top.Device, top.Score, policy.Device)
				for _, r := range ranked {
					t.Logf("  %s: score=%.2f compared=%d", r.Device, r.Score, r.Compared)
				}
			} else if top.Score < 0.9 {
				t.Errorf("fingerprint ranking correctly identified %q but with a low score %.2f (compared=%d)",
					top.Device, top.Score, top.Compared)
			}
		})
	}
}
