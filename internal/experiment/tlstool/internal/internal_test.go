package internal_test

import (
	"context"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/experiment/tlstool/internal"
	"github.com/ooni/probe-cli/v3/internal/legacy/netx"
	"github.com/ooni/probe-cli/v3/internal/model"
)

var config = internal.DialerConfig{
	Dialer: netx.NewDialer(netx.Config{}),
	Delay:  10,
	SNI:    "dns.google",
}

func dial(t *testing.T, d model.Dialer) {
	td := netx.NewTLSDialer(netx.Config{Dialer: d})
	conn, err := td.DialTLSContext(context.Background(), "tcp", "dns.google:853")
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
}

func TestNewSNISplitterDialer(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}

	dial(t, internal.NewSNISplitterDialer(config))
}

func TestNewThriceSplitterDialer(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}

	dial(t, internal.NewThriceSplitterDialer(config))
}

func TestNewRandomSplitterDialer(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}

	dial(t, internal.NewRandomSplitterDialer(config))
}

func TestNewVanillaDialer(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}

	dial(t, internal.NewVanillaDialer(config))
}
