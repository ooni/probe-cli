package ptx

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/atomicx"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/mockablex"
)

func TestOBFS4DialerWorks(t *testing.T) {
	// This test is 0.3 seconds in my machine, so it's ~fine
	// to run it even when we're in short mode
	o4d := DefaultTestingOBFS4Bridge()
	conn, err := o4d.DialContext(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if conn == nil {
		t.Fatal("expected non-nil conn here")
	}
	if o4d.Name() != "obfs4" {
		t.Fatal("unexpected value returned by Name")
	}
	bridgearg := o4d.AsBridgeArgument()
	expectedbridge := "obfs4 192.95.36.142:443 CDF2E852BF539B82BD10E27E9115A31734E378C2 cert=qUVQ0srL1JI/vO6V6m/24anYXiJD3QP2HgzUKQtQ7GRqqUvs7P+tG43RtAqdhLOALP7DJQ iat-mode=1"
	if bridgearg != expectedbridge {
		t.Fatal("unexpected AsBridgeArgument value", bridgearg)
	}
	conn.Close()
}

func TestOBFS4DialerFailsWithInvalidCert(t *testing.T) {
	o4d := DefaultTestingOBFS4Bridge()
	o4d.Cert = "antani!!!"
	conn, err := o4d.DialContext(context.Background())
	if err == nil || !strings.HasPrefix(err.Error(), "failed to decode cert:") {
		t.Fatal("not the error we expected", err)
	}
	if conn != nil {
		t.Fatal("expected nil conn here")
	}
}

func TestOBFS4DialerFailsWithConnectionErrorAndNoContextExpiration(t *testing.T) {
	expected := errors.New("mocked error")
	o4d := DefaultTestingOBFS4Bridge()
	o4d.UnderlyingDialer = &mockablex.Dialer{
		MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
			return nil, expected
		},
	}
	conn, err := o4d.DialContext(context.Background())
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected", err)
	}
	if conn != nil {
		t.Fatal("expected nil conn here")
	}
}

func TestOBFS4DialerFailsWithConnectionErrorAndContextExpiration(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	expected := errors.New("mocked error")
	o4d := DefaultTestingOBFS4Bridge()
	o4d.UnderlyingDialer = &mockablex.Dialer{
		MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
			// We cancel the context before returning the error, which makes
			// the context cancellation happen before us returning.
			cancel()
			return nil, expected
		},
	}
	conn, err := o4d.DialContext(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected", err)
	}
	if conn != nil {
		t.Fatal("expected nil conn here")
	}
}

// obfs4connwrapper allows us to observe that Close has been called
type obfs4connwrapper struct {
	net.Conn
	called *atomicx.Int64
}

// Close implements net.Conn.Close
func (c *obfs4connwrapper) Close() error {
	c.called.Add(1)
	return c.Conn.Close()
}

func TestOBFS4DialerWorksWithContextExpiration(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	called := &atomicx.Int64{}
	o4d := DefaultTestingOBFS4Bridge()
	o4d.UnderlyingDialer = &mockablex.Dialer{
		MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
			// We cancel the context before returning the error, which makes
			// the context cancellation happen before us returning.
			cancel()
			conn, err := net.Dial(network, address)
			if err != nil {
				return nil, err
			}
			return &obfs4connwrapper{
				Conn:   conn,
				called: called,
			}, nil
		},
	}
	cd, err := o4d.newCancellableDialer()
	if err != nil {
		t.Fatal(err)
	}
	conn, err := cd.dial(ctx, "tcp", o4d.Address)
	if !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected", err)
	}
	if conn != nil {
		t.Fatal("expected nil conn here")
	}
	// The point of returning early when the context expires is
	// to NOT wait for the background goroutine to terminate, but
	// here we wanna observe whether it terminates and whether
	// it calls close. Hence, well, we need to wait :^).
	<-cd.done
	if called.Load() != 1 {
		t.Fatal("the goroutine did not call close")
	}
}
