package quicfp

import (
	"context"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// DiffEntry is one vector's outcome against both targets.
type DiffEntry struct {
	VectorName       string  `json:"vector_name"`
	Category         string  `json:"category"`
	Field            string  `json:"field,omitempty"`
	Permutation      string  `json:"permutation,omitempty"`
	ControlOutcome   Outcome `json:"control_outcome"`
	CandidateOutcome Outcome `json:"candidate_outcome"`
	// Divergent is true exactly when Control and Candidate disagree and
	// both sides produced a confident (non-Unknown, non-Inconsistent)
	// classification. This is the signal worth looking at first: the
	// public control endpoint answered this vector fine, but the
	// candidate path did not (or vice versa), which server-side
	// non-compliance alone cannot explain since the same client
	// behavior was tolerated by Control.
	Divergent bool `json:"divergent"`
}

// DiffResult is the outcome of one RunDifferential call.
type DiffResult struct {
	ControlInput      string       `json:"control_input"`
	CandidateInput    string       `json:"candidate_input"`
	Entries           []*DiffEntry `json:"entries"`
	ControlTestKeys   *TestKeys    `json:"control_test_keys"`
	CandidateTestKeys *TestKeys    `json:"candidate_test_keys"`
	DivergentVectors  []string     `json:"divergent_vectors"`
}

// DefaultControlTarget is a well-provisioned, always-on public QUIC/HTTP3
// endpoint used as the "known good" reference path in the operator-side
// design described in the standalone quicfp repository's METHODOLOGY.md: if
// Control's vectors come back clean but the same vectors fail against
// Candidate, the difference is attributable to something on the path to
// Candidate specifically (or to Candidate's own server-side handling), not
// to your local network being generally broken. cloudflare-quic.com is Cloudflare's own dedicated public
// QUIC/HTTP3 test domain (per the quiche project's own README, "The
// cloudflare-quic.com website can be used for testing and experimentation"),
// making it a more deliberate choice of reference than an arbitrary
// Cloudflare-fronted production site. Override this if you have a more
// suitable reference (e.g. infrastructure you operate yourself).
const DefaultControlTarget = "cloudflare-quic.com"

// runOnce runs the full vector suite once against input, using a fresh
// Measurer/config/socket, and returns its TestKeys. This constructs the
// richer-input Target directly rather than going through the CLI-based
// targetLoader, since this is a programmatic helper, not a CLI invocation.
func runOnce(ctx context.Context, cfg Config, input string, sess model.ExperimentSession) (*TestKeys, error) {
	m := &Measurer{}
	measurement := &model.Measurement{
		MeasurementStartTimeSaved: time.Now(),
	}
	cfgCopy := cfg
	args := &model.ExperimentArgs{
		Measurement: measurement,
		Session:     sess,
		Target:      &Target{Config: &cfgCopy, URL: input},
	}
	if err := m.Run(ctx, args); err != nil {
		return nil, err
	}
	tk, _ := measurement.TestKeys.(*TestKeys)
	return tk, nil
}

// RunDifferential runs the full vector suite against controlInput and
// candidateInput concurrently (so both see roughly the same network
// conditions/time window) and diffs the results. If controlInput is empty,
// DefaultControlTarget is used.
func RunDifferential(ctx context.Context, cfg Config, controlInput, candidateInput string, sess model.ExperimentSession) (*DiffResult, error) {
	if controlInput == "" {
		controlInput = DefaultControlTarget
	}

	var (
		wg                       sync.WaitGroup
		controlTK, candidateTK   *TestKeys
		controlErr, candidateErr error
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		controlTK, controlErr = runOnce(ctx, cfg, controlInput, sess)
	}()
	go func() {
		defer wg.Done()
		candidateTK, candidateErr = runOnce(ctx, cfg, candidateInput, sess)
	}()
	wg.Wait()

	if controlErr != nil {
		return nil, controlErr
	}
	if candidateErr != nil {
		return nil, candidateErr
	}

	controlOutcomes := OutcomeMap(controlTK)
	candidateOutcomes := OutcomeMap(candidateTK)

	// Index candidate vectors by name for field/category metadata and to
	// make sure we report on the union of vector names seen on either
	// side (they should be identical since both runs build the same
	// vector list, but don't assume it).
	meta := make(map[string]*VectorResult, len(candidateTK.Vectors))
	for _, vr := range candidateTK.Vectors {
		meta[vr.Name] = vr
	}
	for _, vr := range controlTK.Vectors {
		if _, ok := meta[vr.Name]; !ok {
			meta[vr.Name] = vr
		}
	}

	result := &DiffResult{
		ControlInput:      controlInput,
		CandidateInput:    candidateInput,
		ControlTestKeys:   controlTK,
		CandidateTestKeys: candidateTK,
	}
	for name, vr := range meta {
		co := controlOutcomes[name]
		ca := candidateOutcomes[name]
		divergent := co != ca &&
			co != OutcomeUnknown && co != OutcomeInconsistent &&
			ca != OutcomeUnknown && ca != OutcomeInconsistent
		entry := &DiffEntry{
			VectorName:       name,
			Category:         vr.Category,
			Field:            vr.Field,
			Permutation:      vr.Permutation,
			ControlOutcome:   co,
			CandidateOutcome: ca,
			Divergent:        divergent,
		}
		result.Entries = append(result.Entries, entry)
		if divergent {
			result.DivergentVectors = append(result.DivergentVectors, name)
		}
	}
	return result, nil
}
