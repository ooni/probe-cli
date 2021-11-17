package modelx

import (
	"context"
	"crypto/tls"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestNewTLSConnectionState(t *testing.T) {
	conn, err := tls.Dial("tcp", "www.google.com:443", nil)
	if err != nil {
		t.Fatal(err)
	}
	state := NewTLSConnectionState(conn.ConnectionState())
	if len(state.PeerCertificates) < 1 {
		t.Fatal("too few certificates")
	}
	if state.Version < tls.VersionSSL30 || state.Version > 0x0304 /*tls.VersionTLS13*/ {
		t.Fatal("unexpected TLS version")
	}
}

func TestMeasurementRoot(t *testing.T) {
	ctx := context.Background()
	if ContextMeasurementRoot(ctx) != nil {
		t.Fatal("unexpected value for ContextMeasurementRoot")
	}
	if ContextMeasurementRootOrDefault(ctx) == nil {
		t.Fatal("unexpected value ContextMeasurementRootOrDefault")
	}
	handler := &dummyHandler{}
	root := &MeasurementRoot{
		Handler:   handler,
		Beginning: time.Time{},
	}
	ctx = WithMeasurementRoot(ctx, root)
	v := ContextMeasurementRoot(ctx)
	if v != root {
		t.Fatal("unexpected ContextMeasurementRoot value")
	}
	v = ContextMeasurementRootOrDefault(ctx)
	if v != root {
		t.Fatal("unexpected ContextMeasurementRoot value")
	}
}

func TestMeasurementRootWithMeasurementRootPanic(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	ctx := context.Background()
	_ = WithMeasurementRoot(ctx, nil)
}

func TestErrWrapperPublicAPI(t *testing.T) {
	child := errors.New("mocked error")
	wrapper := &netxlite.ErrWrapper{
		Failure:    "moobar",
		WrappedErr: child,
	}
	if wrapper.Error() != "moobar" {
		t.Fatal("The Error() method is misbehaving")
	}
	if wrapper.Unwrap() != child {
		t.Fatal("The Unwrap() method is misbehaving")
	}
}

func TestComputeBodySnapSize(t *testing.T) {
	if ComputeBodySnapSize(-1) != math.MaxInt64 {
		t.Fatal("unexpected result")
	}
	if ComputeBodySnapSize(0) != defaultBodySnapSize {
		t.Fatal("unexpected result")
	}
	if ComputeBodySnapSize(127) != 127 {
		t.Fatal("unexpected result")
	}
}
