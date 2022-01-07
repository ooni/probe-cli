package ptx

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/atomicx"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestOBFS4DialerWorks(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
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
	expectedbridge := "obfs4 209.148.46.65:443 74FAD13168806246602538555B5521A0383A1875 cert=ssH+9rP8dG2NLDN2XuFw63hIO/9MNNinLmxQDpVa+7kTOa9/m+tGWT1SmSYpQ9uTBGa6Hw iat-mode=0"
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
	o4d.UnderlyingDialer = &mocks.Dialer{
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
	unexpected := errors.New("mocked error")
	o4d := DefaultTestingOBFS4Bridge()
	sigch := make(chan interface{})
	wg := &sync.WaitGroup{}
	wg.Add(1)
	o4d.UnderlyingDialer = &mocks.Dialer{
		MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
			cancel()
			<-sigch
			wg.Done()
			return nil, unexpected
		},
	}
	conn, err := o4d.DialContext(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected", err)
	}
	if conn != nil {
		t.Fatal("expected nil conn here")
	}
	close(sigch)
	wg.Wait()
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
	o4d.UnderlyingDialer = &mocks.Dialer{
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
