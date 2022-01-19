package ptx

import (
	"context"
	"errors"
	"net"
	"testing"

	sflib "git.torproject.org/pluggable-transports/snowflake.git/v2/client/lib"
	"github.com/ooni/probe-cli/v3/internal/atomicx"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestSnowflakeDialerWorks(t *testing.T) {
	// This test may sadly run for a very long time (~10s)
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	sfd := &SnowflakeDialer{}
	conn, err := sfd.DialContext(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if conn == nil {
		t.Fatal("expected non-nil conn here")
	}
	if sfd.Name() != "snowflake" {
		t.Fatal("the Name function returned an unexpected value")
	}
	expect := "snowflake 192.0.2.3:1 2B280B23E1107BB62ABFC40DDCC8824814F80A72"
	if v := sfd.AsBridgeArgument(); v != expect {
		t.Fatal("AsBridgeArgument returned an unexpected value", v)
	}
	conn.Close()
}

// mockableSnowflakeTransport is a mock for snowflakeTransport
type mockableSnowflakeTransport struct {
	MockDial func() (net.Conn, error)
}

// Dial implements snowflakeTransport.Dial.
func (txp *mockableSnowflakeTransport) Dial() (net.Conn, error) {
	return txp.MockDial()
}

var _ snowflakeTransport = &mockableSnowflakeTransport{}

func TestSnowflakeDialerWorksWithMocks(t *testing.T) {
	sfd := &SnowflakeDialer{
		newClientTransport: func(config sflib.ClientConfig) (snowflakeTransport, error) {
			return &mockableSnowflakeTransport{
				MockDial: func() (net.Conn, error) {
					return &mocks.Conn{
						MockClose: func() error {
							return nil
						},
					}, nil
				},
			}, nil
		},
	}
	conn, err := sfd.DialContext(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if conn == nil {
		t.Fatal("expected non-nil conn here")
	}
	if sfd.Name() != "snowflake" {
		t.Fatal("the Name function returned an unexpected value")
	}
	expect := "snowflake 192.0.2.3:1 2B280B23E1107BB62ABFC40DDCC8824814F80A72"
	if v := sfd.AsBridgeArgument(); v != expect {
		t.Fatal("AsBridgeArgument returned an unexpected value", v)
	}
	conn.Close()
}

func TestSnowflakeDialerCannotCreateTransport(t *testing.T) {
	expected := errors.New("mocked error")
	sfd := &SnowflakeDialer{
		newClientTransport: func(config sflib.ClientConfig) (snowflakeTransport, error) {
			return nil, expected
		},
	}
	conn, err := sfd.DialContext(context.Background())
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected", err)
	}
	if conn != nil {
		t.Fatal("expected nil conn here")
	}
}

func TestSnowflakeDialerCannotCreateConnWithNoContextExpiration(t *testing.T) {
	expected := errors.New("mocked error")
	sfd := &SnowflakeDialer{
		newClientTransport: func(config sflib.ClientConfig) (snowflakeTransport, error) {
			return &mockableSnowflakeTransport{
				MockDial: func() (net.Conn, error) {
					return nil, expected
				},
			}, nil
		},
	}
	conn, err := sfd.DialContext(context.Background())
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected", err)
	}
	if conn != nil {
		t.Fatal("expected nil conn here")
	}
}

func TestSnowflakeDialerCannotCreateConnWithContextExpiration(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	expected := errors.New("mocked error")
	sfd := &SnowflakeDialer{
		newClientTransport: func(config sflib.ClientConfig) (snowflakeTransport, error) {
			return &mockableSnowflakeTransport{
				MockDial: func() (net.Conn, error) {
					cancel() // before returning to the caller
					return nil, expected
				},
			}, nil
		},
	}
	conn, err := sfd.DialContext(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected", err)
	}
	if conn != nil {
		t.Fatal("expected nil conn here")
	}
}

func TestSnowflakeDialerWorksWithWithCancelledContext(t *testing.T) {
	called := &atomicx.Int64{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sfd := &SnowflakeDialer{
		newClientTransport: func(config sflib.ClientConfig) (snowflakeTransport, error) {
			return &mockableSnowflakeTransport{
				MockDial: func() (net.Conn, error) {
					cancel() // cause a cancel before we can really have a conn
					return &mocks.Conn{
						MockClose: func() error {
							called.Add(1)
							return nil
						},
					}, nil
				},
			}, nil
		},
	}
	conn, done, err := sfd.dialContext(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected", err)
	}
	if conn != nil {
		t.Fatal("expected nil conn here")
	}
	// synchronize with the end of the inner goroutine
	<-done
	if called.Load() != 1 {
		t.Fatal("the goroutine did not call close")
	}
}
