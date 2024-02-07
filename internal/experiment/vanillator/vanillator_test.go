package vanillator

import (
	"context"
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/legacy/mockable"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/tunnel"
	"github.com/ooni/probe-cli/v3/internal/tunnel/mocks"
)

// Implementation note: this file is written with easy diffing with respect
// to internal/experiment/torsf/torsf_test.go in mind.
//
// We may want to have a single implementation for both nettests in the future.

func TestExperimentNameAndVersion(t *testing.T) {
	m := NewExperimentMeasurer(Config{})
	if m.ExperimentName() != "vanilla_tor" {
		t.Fatal("invalid experiment name")
	}
	if m.ExperimentVersion() != "0.3.0" {
		t.Fatal("invalid experiment version")
	}
}

func TestSuccessWithMockedTunnelStart(t *testing.T) {
	bootstrapTime := 3 * time.Second
	called := &atomic.Int64{}
	m := &Measurer{
		config: Config{},
		mockStartTunnel: func(
			ctx context.Context, config *tunnel.Config) (tunnel.Tunnel, tunnel.DebugInfo, error) {
			// run for some time so we also exercise printing progress.
			time.Sleep(bootstrapTime)
			return &mocks.Tunnel{
					MockBootstrapTime: func() time.Duration {
						return bootstrapTime
					},
					MockStop: func() {
						called.Add(1)
					},
				}, tunnel.DebugInfo{
					Name:        "tor",
					LogFilePath: filepath.Join("testdata", "tor.log"),
				}, nil
		},
	}
	ctx := context.Background()
	measurement := &model.Measurement{}
	sess := &mockable.Session{
		MockableLogger: model.DiscardLogger,
	}
	callbacks := &model.PrinterCallbacks{
		Logger: model.DiscardLogger,
	}
	args := &model.ExperimentArgs{
		Callbacks:   callbacks,
		Measurement: measurement,
		Session:     sess,
	}
	if err := m.Run(ctx, args); err != nil {
		t.Fatal(err)
	}
	if called.Load() != 1 {
		t.Fatal("stop was not called")
	}
	tk := measurement.TestKeys.(*TestKeys)
	if tk.BootstrapTime != bootstrapTime.Seconds() {
		t.Fatal("unexpected bootstrap time")
	}
	if tk.Error != nil {
		t.Fatal("unexpected error")
	}
	if tk.Failure != nil {
		t.Fatal("unexpected failure")
	}
	if !tk.Success {
		t.Fatal("unexpected success value")
	}
	if tk.Timeout != maxRuntime.Seconds() {
		t.Fatal("unexpected timeout")
	}
	if count := len(tk.TorLogs); count != 9 {
		t.Fatal("unexpected length of tor logs", count)
	}
	if tk.TorProgress != 100 {
		t.Fatal("unexpected tor progress")
	}
	if tk.TorProgressTag != "done" {
		t.Fatal("unexpected tor progress tag")
	}
	if tk.TorProgressSummary != "Done" {
		t.Fatal("unexpected tor progress tag")
	}
	if tk.TransportName != "vanilla" {
		t.Fatal("invalid transport name")
	}
}

func TestWithCancelledContext(t *testing.T) {
	// This test calls the real tunnel.Start function so we cover
	// it but fails immediately because of the cancelled ctx.
	m := &Measurer{
		config: Config{},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // fail immediately
	measurement := &model.Measurement{}
	sess := &mockable.Session{
		MockableLogger: model.DiscardLogger,
	}
	callbacks := &model.PrinterCallbacks{
		Logger: model.DiscardLogger,
	}
	args := &model.ExperimentArgs{
		Callbacks:   callbacks,
		Measurement: measurement,
		Session:     sess,
	}
	if err := m.Run(ctx, args); err != nil {
		t.Fatal(err)
	}
	tk := measurement.TestKeys.(*TestKeys)
	if tk.BootstrapTime != 0 {
		t.Fatal("unexpected bootstrap time")
	}
	if tk.Error == nil || *tk.Error != "unknown-error" {
		t.Fatal("unexpected error")
	}
	if *tk.Failure != "interrupted" {
		t.Fatal("unexpected failure")
	}
	if tk.Success {
		t.Fatal("unexpected success value")
	}
	if tk.Timeout != maxRuntime.Seconds() {
		t.Fatal("unexpected timeout")
	}
	if len(tk.TorLogs) != 0 {
		t.Fatal("unexpected length of tor logs")
	}
	if tk.TorProgress != 0 {
		t.Fatal("unexpected tor progress")
	}
	if tk.TorProgressTag != "" {
		t.Fatal("unexpected tor progress tag")
	}
	if tk.TorProgressSummary != "" {
		t.Fatal("unexpected tor progress tag")
	}
	if tk.TransportName != "vanilla" {
		t.Fatal("invalid transport name")
	}
}

func TestFailureToStartTunnel(t *testing.T) {
	expected := context.DeadlineExceeded // error occurring on bootstrap timeout
	m := &Measurer{
		config: Config{},
		mockStartTunnel: func(
			ctx context.Context, config *tunnel.Config) (tunnel.Tunnel, tunnel.DebugInfo, error) {
			return nil,
				tunnel.DebugInfo{
					Name:        "tor",
					LogFilePath: filepath.Join("testdata", "partial.log"),
				}, expected
		},
	}
	ctx := context.Background()
	measurement := &model.Measurement{}
	sess := &mockable.Session{
		MockableLogger: model.DiscardLogger,
	}
	callbacks := &model.PrinterCallbacks{
		Logger: model.DiscardLogger,
	}
	args := &model.ExperimentArgs{
		Callbacks:   callbacks,
		Measurement: measurement,
		Session:     sess,
	}
	if err := m.Run(ctx, args); err != nil {
		t.Fatal(err)
	}
	tk := measurement.TestKeys.(*TestKeys)
	if tk.BootstrapTime != 0 {
		t.Fatal("unexpected bootstrap time")
	}
	if tk.Error == nil || *tk.Error != "timeout-reached" {
		t.Fatal("unexpected error")
	}
	if tk.Failure == nil {
		t.Fatal("unexpectedly nil failure string")
	}
	if *tk.Failure != "generic_timeout_error" {
		t.Fatal("unexpected failure string", *tk.Failure)
	}
	if tk.Success {
		t.Fatal("unexpected success value")
	}
	if tk.Timeout != maxRuntime.Seconds() {
		t.Fatal("unexpected timeout")
	}
	if count := len(tk.TorLogs); count != 6 {
		t.Fatal("unexpected length of tor logs", count)
	}
	if tk.TorProgress != 15 {
		t.Fatal("unexpected tor progress")
	}
	if tk.TorProgressTag != "handshake_done" {
		t.Fatal("unexpected tor progress tag")
	}
	if tk.TorProgressSummary != "Handshake with a relay done" {
		t.Fatal("unexpected tor progress tag")
	}
	if tk.TransportName != "vanilla" {
		t.Fatal("invalid transport name")
	}
}

func TestFailureNoTorBinary(t *testing.T) {
	t.Run("with mocked startTunnel", func(t *testing.T) {
		expected := tunnel.ErrCannotFindTorBinary
		m := &Measurer{
			config: Config{},
			mockStartTunnel: func(ctx context.Context, config *tunnel.Config) (tunnel.Tunnel, tunnel.DebugInfo, error) {
				return nil,
					tunnel.DebugInfo{
						Name:        "tor",
						LogFilePath: filepath.Join("testdata", "partial.log"),
					}, expected
			},
		}
		ctx := context.Background()
		measurement := &model.Measurement{}
		sess := &mockable.Session{
			MockableLogger: model.DiscardLogger,
		}
		callbacks := &model.PrinterCallbacks{
			Logger: model.DiscardLogger,
		}
		args := &model.ExperimentArgs{
			Callbacks:   callbacks,
			Measurement: measurement,
			Session:     sess,
		}
		if err := m.Run(ctx, args); !errors.Is(err, expected) {
			t.Fatal("unexpected error")
		}
		tk := measurement.TestKeys.(*TestKeys)
		if tk.BootstrapTime != 0 {
			t.Fatal("unexpected bootstrap time")
		}
		if tk.Error == nil || *tk.Error != "unknown-error" {
			t.Fatal("unexpected error")
		}
		if tk.Failure == nil {
			t.Fatal("unexpectedly nil failure string")
		}
		if *tk.Failure != "unknown_failure: tunnel: cannot find tor binary" {
			t.Fatal("unexpected failure string", *tk.Failure)
		}
		if tk.Success {
			t.Fatal("unexpected success value")
		}
		if !tk.cannotFindTorBinary {
			t.Fatal("unexpected cannotFindTorBinary values")
		}
		if tk.Timeout != maxRuntime.Seconds() {
			t.Fatal("unexpected timeout")
		}
		if count := len(tk.TorLogs); count != 6 {
			t.Fatal("unexpected length of tor logs", count)
		}
		if tk.TorProgress != 15 {
			t.Fatal("unexpected tor progress")
		}
		if tk.TorProgressTag != "handshake_done" {
			t.Fatal("unexpected tor progress tag")
		}
		if tk.TorProgressSummary != "Handshake with a relay done" {
			t.Fatal("unexpected tor progress tag")
		}
		if tk.TransportName != "vanilla" {
			t.Fatal("invalid transport name")
		}

	})
}

func TestMeasurementSummaryKeys(t *testing.T) {
	t.Run("in case of success", func(t *testing.T) {
		measurement := &model.Measurement{
			TestKeys: &TestKeys{
				Failure: nil,
			},
		}
		sk := measurement.TestKeys.(*TestKeys).MeasurementSummaryKeys()
		rsk := sk.(*SummaryKeys)
		if rsk.IsAnomaly {
			t.Fatal("expected no anomaly here")
		}
		if rsk.IsAnomaly != sk.Anomaly() {
			t.Fatal("invalid Anomaly()")
		}
	})

	t.Run("in case of failure", func(t *testing.T) {
		failure := "generic_timeout_error"
		measurement := &model.Measurement{
			TestKeys: &TestKeys{
				Failure: &failure,
			},
		}
		sk := measurement.TestKeys.(*TestKeys).MeasurementSummaryKeys()
		rsk := sk.(*SummaryKeys)
		if !rsk.IsAnomaly {
			t.Fatal("expected anomaly here")
		}
		if rsk.IsAnomaly != sk.Anomaly() {
			t.Fatal("invalid Anomaly()")
		}
	})
}
