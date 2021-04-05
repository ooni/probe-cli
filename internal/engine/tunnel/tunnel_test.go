package tunnel_test

import (
	"context"
	"errors"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/internal/mockable"
	"github.com/ooni/probe-cli/v3/internal/engine/tunnel"
)

func TestStartNoTunnel(t *testing.T) {
	ctx := context.Background()
	tun, err := tunnel.Start(ctx, &tunnel.Config{
		Name: "",
		Session: &mockable.Session{
			MockableLogger: log.Log,
		},
	})
	if !errors.Is(err, tunnel.ErrUnsupportedTunnelName) {
		t.Fatal("not the error we expected", err)
	}
	if tun != nil {
		t.Fatal("expected nil tunnel here")
	}
}

func TestStartPsiphonWithCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // fail immediately
	tun, err := tunnel.Start(ctx, &tunnel.Config{
		Name: "psiphon",
		Session: &mockable.Session{
			MockableLogger: log.Log,
		},
		TunnelDir: "testdata",
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected")
	}
	if tun != nil {
		t.Fatal("expected nil tunnel here")
	}
}

func TestStartTorWithCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // fail immediately
	tun, err := tunnel.Start(ctx, &tunnel.Config{
		Name: "tor",
		Session: &mockable.Session{
			MockableLogger: log.Log,
		},
		TunnelDir: "testdata",
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected")
	}
	if tun != nil {
		t.Fatal("expected nil tunnel here")
	}
}

func TestStartInvalidTunnel(t *testing.T) {
	ctx := context.Background()
	tun, err := tunnel.Start(ctx, &tunnel.Config{
		Name: "antani",
		Session: &mockable.Session{
			MockableLogger: log.Log,
		},
		TunnelDir: "testdata",
	})
	if !errors.Is(err, tunnel.ErrUnsupportedTunnelName) {
		t.Fatal("not the error we expected")
	}
	if tun != nil {
		t.Fatal("expected nil tunnel here")
	}
}
