package torsf

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/atomicx"
	"github.com/ooni/probe-cli/v3/internal/engine/mockable"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/ptx"
	"github.com/ooni/probe-cli/v3/internal/tunnel"
	"github.com/ooni/probe-cli/v3/internal/tunnel/mocks"
)

// Implementation note: this file is written with easy diffing with respect
// to internal/engine/experiment/vanillator/vanillator_test.go in mind.
//
// We may want to have a single implementation for both nettests in the future.

func TestExperimentNameAndVersion(t *testing.T) {
	m := NewExperimentMeasurer(Config{})
	if m.ExperimentName() != "torsf" {
		t.Fatal("invalid experiment name")
	}
	if m.ExperimentVersion() != "0.3.0" {
		t.Fatal("invalid experiment version")
	}
}

func TestFailureWithInvalidRendezvousMethod(t *testing.T) {
	m := &Measurer{
		config: Config{
			DisablePersistentDatadir: false,
			DisableProgress:          false,
			RendezvousMethod:         "antani",
		},
		mockStartTunnel: nil,
	}
	ctx := context.Background()
	measurement := &model.Measurement{}
	sess := &mockable.Session{
		MockableLogger: model.DiscardLogger,
	}
	callbacks := &model.PrinterCallbacks{
		Logger: model.DiscardLogger,
	}
	err := m.Run(ctx, sess, measurement, callbacks)
	if !errors.Is(err, ptx.ErrSnowflakeNoSuchRendezvousMethod) {
		t.Fatal("unexpected error", err)
	}
	if measurement.TestKeys != nil {
		t.Fatal("expected nil test keys")
	}
}

func TestFailureToStartPTXListener(t *testing.T) {
	expected := errors.New("mocked error")
	m := &Measurer{
		config: Config{},
		mockStartListener: func() error {
			return expected
		},
	}
	ctx := context.Background()
	measurement := &model.Measurement{}
	sess := &mockable.Session{}
	callbacks := &model.PrinterCallbacks{
		Logger: model.DiscardLogger,
	}
	if err := m.Run(ctx, sess, measurement, callbacks); !errors.Is(err, expected) {
		t.Fatal("not the error we expected", err)
	}
	if tk := measurement.TestKeys; tk != nil {
		t.Fatal("expected nil bootstrap time here")
	}
}

func TestSuccessWithMockedTunnelStart(t *testing.T) {
	bootstrapTime := 3 * time.Second
	called := &atomicx.Int64{}
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
	if err := m.Run(ctx, sess, measurement, callbacks); err != nil {
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
	if !tk.PersistentDatadir {
		t.Fatal("unexpected persistent data dir")
	}
	if tk.RendezvousMethod != "domain_fronting" {
		t.Fatal("unexpected rendezvous method")
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
	if tk.TransportName != "snowflake" {
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
	if err := m.Run(ctx, sess, measurement, callbacks); err != nil {
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
	if !tk.PersistentDatadir {
		t.Fatal("unexpected persistent data dir")
	}
	if tk.RendezvousMethod != "domain_fronting" {
		t.Fatal("unexpected rendezvous method")
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
	if tk.TransportName != "snowflake" {
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
	if err := m.Run(ctx, sess, measurement, callbacks); err != nil {
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
	if !tk.PersistentDatadir {
		t.Fatal("unexpected persistent datadir")
	}
	if tk.RendezvousMethod != "domain_fronting" {
		t.Fatal("unexpected rendezvous method")
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
	if tk.TransportName != "snowflake" {
		t.Fatal("invalid transport name")
	}
}

func TestBaseTunnelDir(t *testing.T) {
	t.Run("without persistent data dir", func(t *testing.T) {
		m := &Measurer{
			config: Config{
				DisablePersistentDatadir: true,
			},
		}
		sess := &mockable.Session{
			MockableTunnelDir: "a",
			MockableTempDir:   "b",
		}
		dir := m.baseTunnelDir(sess)
		if dir != "b" {
			t.Fatal("unexpected base tunnel dir", dir)
		}
	})

	t.Run("with persistent data dir", func(t *testing.T) {
		m := &Measurer{
			config: Config{
				DisablePersistentDatadir: false,
			},
		}
		sess := &mockable.Session{
			MockableTunnelDir: "a",
			MockableTempDir:   "b",
		}
		dir := m.baseTunnelDir(sess)
		if dir != "a" {
			t.Fatal("unexpected base tunnel dir", dir)
		}
	})
}

func TestGetSummaryKeys(t *testing.T) {
	t.Run("in case of untyped nil TestKeys", func(t *testing.T) {
		measurement := &model.Measurement{
			TestKeys: nil,
		}
		m := &Measurer{}
		_, err := m.GetSummaryKeys(measurement)
		if !errors.Is(err, errInvalidTestKeysType) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("in case of typed nil TestKeys", func(t *testing.T) {
		var tk *TestKeys
		measurement := &model.Measurement{
			TestKeys: tk,
		}
		m := &Measurer{}
		_, err := m.GetSummaryKeys(measurement)
		if !errors.Is(err, errNilTestKeys) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("in case of invalid TestKeys type", func(t *testing.T) {
		measurement := &model.Measurement{
			TestKeys: make(chan int),
		}
		m := &Measurer{}
		_, err := m.GetSummaryKeys(measurement)
		if !errors.Is(err, errInvalidTestKeysType) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("in case of success", func(t *testing.T) {
		measurement := &model.Measurement{
			TestKeys: &TestKeys{
				Failure: nil,
			},
		}
		m := &Measurer{}
		sk, err := m.GetSummaryKeys(measurement)
		if err != nil {
			t.Fatal(err)
		}
		rsk := sk.(SummaryKeys)
		if rsk.IsAnomaly {
			t.Fatal("expected no anomaly here")
		}
	})

	t.Run("in case of failure", func(t *testing.T) {
		failure := "generic_timeout_error"
		measurement := &model.Measurement{
			TestKeys: &TestKeys{
				Failure: &failure,
			},
		}
		m := &Measurer{}
		sk, err := m.GetSummaryKeys(measurement)
		if err != nil {
			t.Fatal(err)
		}
		rsk := sk.(SummaryKeys)
		if !rsk.IsAnomaly {
			t.Fatal("expected anomaly here")
		}
	})
}
