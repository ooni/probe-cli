package measurexlite_test

import (
	"context"
	"crypto/tls"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
)

func TestCountSystemResolverBytes(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}

	// create the trace
	tx := measurexlite.NewTrace(0, time.Now())

	// create the context
	ctx := context.Background()

	// add and register session byte counter
	sbc := bytecounter.New()
	ctx = bytecounter.WithSessionByteCounter(ctx, sbc)

	// add and register experiment byte counter
	ebc := bytecounter.New()
	ctx = bytecounter.WithExperimentByteCounter(ctx, ebc)

	// create system resolver
	reso := tx.NewStdlibResolver(log.Log)
	defer reso.CloseIdleConnections()

	// run a lookup
	addrs, err := reso.LookupHost(ctx, "www.example.com")

	// make sure we didn't fail
	if err != nil {
		t.Fatal(err)
	}

	// make sure we resolved addresses
	if len(addrs) <= 0 {
		t.Fatal("expected at least one address")
	}

	// make sure we received something
	if sbc.Received.Load() <= 0 {
		t.Fatal("sbs.Received.Load() returned zero or less")
	}
	if ebc.Received.Load() <= 0 {
		t.Fatal("ebc.Received.Load() returned zero or less")
	}

	// make sure we send something
	if sbc.Sent.Load() <= 0 {
		t.Fatal("sbs.Sent.Load() returned zero or less")
	}
	if ebc.Sent.Load() <= 0 {
		t.Fatal("ebc.Sent.Load() returned zero or less")
	}
}

func TestCountConnBytes(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}

	// create the trace
	tx := measurexlite.NewTrace(0, time.Now())

	// create the context
	ctx := context.Background()

	// add and register session byte counter
	sbc := bytecounter.New()
	ctx = bytecounter.WithSessionByteCounter(ctx, sbc)

	// add and register experiment byte counter
	ebc := bytecounter.New()
	ctx = bytecounter.WithExperimentByteCounter(ctx, ebc)

	// create dialer
	reso := tx.NewDialerWithoutResolver(log.Log)

	// run a lookup
	conn, err := reso.DialContext(ctx, "tcp", "8.8.8.8:443")
	defer measurexlite.MaybeClose(conn)

	// make sure we didn't fail
	if err != nil {
		t.Fatal(err)
	}

	// create the handshaker
	thx := tx.NewTLSHandshakerStdlib(log.Log)

	// handshake
	tconn, err := thx.Handshake(ctx, conn, &tls.Config{ServerName: "dns.google"})
	defer measurexlite.MaybeClose(tconn)

	// make sure we didn't fail
	if err != nil {
		t.Fatal(err)
	}

	// make sure we received something
	if sbc.Received.Load() <= 0 {
		t.Fatal("sbs.Received.Load() returned zero or less")
	}
	if ebc.Received.Load() <= 0 {
		t.Fatal("ebc.Received.Load() returned zero or less")
	}

	// make sure we send something
	if sbc.Sent.Load() <= 0 {
		t.Fatal("sbs.Sent.Load() returned zero or less")
	}
	if ebc.Sent.Load() <= 0 {
		t.Fatal("ebc.Sent.Load() returned zero or less")
	}
}
