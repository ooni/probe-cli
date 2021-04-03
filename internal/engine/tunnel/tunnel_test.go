package tunnel

import (
	"context"
	"errors"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/internal/mockable"
)

func TestStartNoTunnel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	tunnel, err := Start(ctx, &Config{
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

func TestStartPsiphonTunnel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	tunnel, err := Start(ctx, &Config{
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

func TestStartTorTunnel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	tunnel, err := Start(ctx, &Config{
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

func TestStartInvalidTunnel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	tunnel, err := Start(ctx, &Config{
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
