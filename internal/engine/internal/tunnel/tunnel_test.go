package tunnel_test

import (
	"context"
	"errors"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/internal/mockable"
	"github.com/ooni/probe-cli/v3/internal/engine/internal/tunnel"
)

func TestNoTunnel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	tunnel, err := tunnel.Start(ctx, tunnel.Config{
		Name: "",
		Session: &mockable.Session{
			MockableLogger: log.Log,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if tunnel != nil {
		t.Fatal("expected nil tunnel here")
	}
}

func TestPsiphonTunnel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	tunnel, err := tunnel.Start(ctx, tunnel.Config{
		Name: "psiphon",
		Session: &mockable.Session{
			MockableLogger: log.Log,
		},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected")
	}
	if tunnel != nil {
		t.Fatal("expected nil tunnel here")
	}
}

func TestTorTunnel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	tunnel, err := tunnel.Start(ctx, tunnel.Config{
		Name: "tor",
		Session: &mockable.Session{
			MockableLogger: log.Log,
		},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected")
	}
	if tunnel != nil {
		t.Fatal("expected nil tunnel here")
	}
}

func TestInvalidTunnel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	tunnel, err := tunnel.Start(ctx, tunnel.Config{
		Name: "antani",
		Session: &mockable.Session{
			MockableLogger: log.Log,
		},
	})
	if err == nil || err.Error() != "unsupported tunnel" {
		t.Fatal("not the error we expected")
	}
	t.Log(tunnel)
	if tunnel != nil {
		t.Fatal("expected nil tunnel here")
	}
}
