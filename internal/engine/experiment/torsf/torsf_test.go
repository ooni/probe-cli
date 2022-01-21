package torsf

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/mockable"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/tunnel"
)

func TestExperimentNameAndVersion(t *testing.T) {
	m := NewExperimentMeasurer(Config{})
	if m.ExperimentName() != "torsf" {
		t.Fatal("invalid experiment name")
	}
	if m.ExperimentVersion() != "0.1.1" {
		t.Fatal("invalid experiment version")
	}
}

// mockedTunnel is a mocked tunnel.
type mockedTunnel struct {
	bootstrapTime time.Duration
	proxyURL      *url.URL
}

// BootstrapTime implements Tunnel.BootstrapTime.
func (mt *mockedTunnel) BootstrapTime() time.Duration {
	return mt.bootstrapTime
}

// SOCKS5ProxyURL implements Tunnel.SOCKS5ProxyURL.
func (mt *mockedTunnel) SOCKS5ProxyURL() *url.URL {
	return mt.proxyURL
}

// Stop implements Tunnel.Stop.
func (mt *mockedTunnel) Stop() {
	// nothing
}

func TestSuccessWithMockedTunnelStart(t *testing.T) {
	bootstrapTime := 400 * time.Millisecond
	m := &Measurer{
		config: Config{},
		mockStartTunnel: func(ctx context.Context, config *tunnel.Config) (tunnel.Tunnel, error) {
			// run for some time so we also exercise printing progress.
			time.Sleep(bootstrapTime)
			return &mockedTunnel{
				bootstrapTime: time.Duration(bootstrapTime),
			}, nil
		},
	}
	ctx := context.Background()
	measurement := &model.Measurement{}
	sess := &mockable.Session{}
	callbacks := &model.PrinterCallbacks{
		Logger: log.Log,
	}
	if err := m.Run(ctx, sess, measurement, callbacks); err != nil {
		t.Fatal(err)
	}
	tk := measurement.TestKeys.(*TestKeys)
	if tk.BootstrapTime != bootstrapTime.Seconds() {
		t.Fatal("unexpected bootstrap time")
	}
}

func TestFailureToStartTunnel(t *testing.T) {
	expected := errors.New("mocked error")
	m := &Measurer{
		config: Config{},
		mockStartTunnel: func(ctx context.Context, config *tunnel.Config) (tunnel.Tunnel, error) {
			return nil, expected
		},
	}
	ctx := context.Background()
	measurement := &model.Measurement{}
	sess := &mockable.Session{}
	callbacks := &model.PrinterCallbacks{
		Logger: log.Log,
	}
	if err := m.Run(ctx, sess, measurement, callbacks); err != nil {
		t.Fatal(err)
	}
	tk := measurement.TestKeys.(*TestKeys)
	if tk.BootstrapTime != 0 {
		t.Fatal("unexpected bootstrap time")
	}
	if tk.Failure == nil {
		t.Fatal("unexpectedly nil failure string")
	}
	if *tk.Failure != "unknown_failure: mocked error" {
		t.Fatal("unexpected failure string", *tk.Failure)
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
		Logger: log.Log,
	}
	if err := m.Run(ctx, sess, measurement, callbacks); !errors.Is(err, expected) {
		t.Fatal("not the error we expected", err)
	}
	tk := measurement.TestKeys.(*TestKeys)
	if tk.BootstrapTime != 0 {
		t.Fatal("unexpected bootstrap time")
	}
	if tk.Failure == nil {
		t.Fatal("unexpectedly nil failure string")
	}
	if *tk.Failure != "unknown_failure: mocked error" {
		t.Fatal("unexpected failure string", *tk.Failure)
	}
}

func TestStartWithCancelledContext(t *testing.T) {
	m := &Measurer{config: Config{}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // fail immediately
	measurement := &model.Measurement{}
	sess := &mockable.Session{}
	callbacks := &model.PrinterCallbacks{
		Logger: log.Log,
	}
	if err := m.Run(ctx, sess, measurement, callbacks); err != nil {
		t.Fatal(err)
	}
	tk := measurement.TestKeys.(*TestKeys)
	if tk.BootstrapTime != 0 {
		t.Fatal("unexpected bootstrap time")
	}
	if tk.Failure == nil {
		t.Fatal("unexpected nil failure")
	}
	if *tk.Failure != "interrupted" {
		t.Fatal("unexpected failure string", *tk.Failure)
	}
}

func TestGetSummaryKeys(t *testing.T) {
	measurement := &model.Measurement{}
	m := &Measurer{}
	sk, err := m.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	rsk := sk.(SummaryKeys)
	if rsk.IsAnomaly {
		t.Fatal("expected no anomaly here")
	}
}
