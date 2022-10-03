package tunnel_test

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/tunnel"
)

func TestStartNoTunnel(t *testing.T) {
	ctx := context.Background()
	tun, _, err := tunnel.Start(ctx, &tunnel.Config{
		Name:    "",
		Session: &tunnel.MockableSession{},
	})
	if !errors.Is(err, tunnel.ErrUnsupportedTunnelName) {
		t.Fatal("not the error we expected", err)
	}
	if tun != nil {
		t.Fatal("expected nil tunnel here")
	}
}

func TestStartPsiphonWithCancelledContext(t *testing.T) {
	if strings.HasPrefix(runtime.Version(), "go1.19") {
		t.Skip("TODO(https://github.com/ooni/probe/issues/2222)")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // fail immediately
	tun, _, err := tunnel.Start(ctx, &tunnel.Config{
		Name:      "psiphon",
		Session:   &tunnel.MockableSession{},
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
	tun, _, err := tunnel.Start(ctx, &tunnel.Config{
		Name:      "tor",
		Session:   &tunnel.MockableSession{},
		TunnelDir: "testdata",
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected")
	}
	if tun != nil {
		t.Fatal("expected nil tunnel here")
	}
}

func TestStartTorsfWithCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // fail immediately
	tun, _, err := tunnel.Start(ctx, &tunnel.Config{
		Name:      "torsf",
		Session:   &tunnel.MockableSession{},
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
	tun, _, err := tunnel.Start(ctx, &tunnel.Config{
		Name:      "antani",
		Session:   &tunnel.MockableSession{},
		TunnelDir: "testdata",
	})
	if !errors.Is(err, tunnel.ErrUnsupportedTunnelName) {
		t.Fatal("not the error we expected")
	}
	if tun != nil {
		t.Fatal("expected nil tunnel here")
	}
}
