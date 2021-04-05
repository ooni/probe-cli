package tunnel_test

import (
	"context"
	"errors"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/engine/tunnel"
)

func TestPsiphonStartWithCancelledContext(t *testing.T) {
	// TODO(bassosimone): this test can use a mockable session so we
	// can move it inside of the internal tests.
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // fail immediately
	sess, err := engine.NewSession(ctx, engine.SessionConfig{
		Logger:          log.Log,
		SoftwareName:    "miniooni",
		SoftwareVersion: "0.1.0-dev",
		TunnelDir:       "testdata",
	})
	if err != nil {
		t.Fatal(err)
	}
	tunnel, err := tunnel.Start(ctx, &tunnel.Config{
		Name:      "psiphon",
		Session:   sess,
		TunnelDir: "testdata",
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected")
	}
	if tunnel != nil {
		t.Fatal("expected nil tunnel here")
	}
}

func TestPsiphonStartStop(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	ctx := context.Background()
	sess, err := engine.NewSession(ctx, engine.SessionConfig{
		Logger:          log.Log,
		SoftwareName:    "ooniprobe-engine",
		SoftwareVersion: "0.0.1",
		TunnelDir:       "testdata",
	})
	if err != nil {
		t.Fatal(err)
	}
	tunnel, err := tunnel.Start(context.Background(), &tunnel.Config{
		Name:      "psiphon",
		Session:   sess,
		TunnelDir: "testdata",
	})
	if err != nil {
		t.Fatal(err)
	}
	if tunnel.SOCKS5ProxyURL() == nil {
		t.Fatal("expected non nil URL here")
	}
	if tunnel.BootstrapTime() <= 0 {
		t.Fatal("expected positive bootstrap time here")
	}
	tunnel.Stop()
}
