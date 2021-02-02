package sessionresolver_test

import (
	"context"
	"strings"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/engine/internal/sessionresolver"
	"github.com/ooni/probe-cli/v3/internal/engine/netx"
)

func TestFallbackWorks(t *testing.T) {
	reso := sessionresolver.New(netx.Config{})
	defer reso.CloseIdleConnections()
	if reso.Network() != "sessionresolver" {
		t.Fatal("unexpected Network")
	}
	if reso.Address() != "" {
		t.Fatal("unexpected Address")
	}
	addrs, err := reso.LookupHost(context.Background(), "antani.ooni.nu")
	if err == nil || !strings.HasSuffix(err.Error(), "no such host") {
		t.Fatal("not the error we expected")
	}
	if addrs != nil {
		t.Fatal("expected nil addrs here")
	}
	if reso.PrimaryFailure.Load() != 1 || reso.FallbackFailure.Load() != 1 {
		t.Fatal("not the counters we expected to see here")
	}
}
