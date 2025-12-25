package ootunnel_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/engine/internal/ootunnel"
)

func TestBrokerNewTunnelTorAndWipeStateDir(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	broker := &ootunnel.Broker{}
	statedir := filepath.Join("testdata", "tor-ephemeral")
	conf := &ootunnel.Config{
		DeleteStateDirOnClose: true,
		Name:                  ootunnel.Tor,
		StateDir:              statedir,
	}
	ctx := context.Background()
	tun, err := broker.NewTunnel(ctx, conf)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(tun.BootstrapTime())
	t.Log(tun.ProxyURL().String())
	t.Log(tun.StateDir())
	tun.Close() // sync so we can observe its effect
	if _, err := os.Stat(statedir); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("not the error we expected", err)
	}
}

func TestBrokerNewTunnelTorAndKeepStateDir(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	broker := &ootunnel.Broker{}
	statedir := filepath.Join("testdata", "tor-persist")
	conf := &ootunnel.Config{
		Name:     ootunnel.Tor,
		StateDir: statedir,
	}
	ctx := context.Background()
	runOnce := func() {
		tun, err := broker.NewTunnel(ctx, conf)
		if err != nil {
			t.Fatal(err)
		}
		t.Log(tun.BootstrapTime())
		t.Log(tun.ProxyURL().String())
		t.Log(tun.StateDir())
		tun.Close() // sync so we can observe its effect
		if _, err := os.Stat(statedir); !errors.Is(err, nil) {
			t.Fatal("not the error we expected", err)
		}
	}
	for i := 0; i < 3; i++ {
		runOnce()
	}
	os.RemoveAll(statedir)
}

func TestBrokerNewManagedTunnelTor(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	broker := &ootunnel.Broker{}
	statedir := filepath.Join("testdata", "tor-managed")
	conf := &ootunnel.Config{
		DeleteStateDirOnClose: true,
		Name:                  ootunnel.Tor,
		StateDir:              statedir,
	}
	ctx := context.Background()
	if err := broker.NewManagedTunnel(ctx, conf); err != nil {
		t.Fatal(err)
	}
	tun, found := broker.GetManagedTunnel(ootunnel.Tor)
	if found != true {
		t.Fatal("not found")
	}
	t.Log(tun.BootstrapTime())
	t.Log(tun.ProxyURL().String())
	t.Log(tun.StateDir())
	broker.Close() // sync so we can observe its effect
	if _, err := os.Stat(statedir); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("not the error we expected", err)
	}
}
