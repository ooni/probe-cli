package ptx

import (
	"context"
	"errors"
	"net"
	"sync/atomic"
	"testing"

	sflib "git.torproject.org/pluggable-transports/snowflake.git/v2/client/lib"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestSnowflakeMethodDomainFronting(t *testing.T) {
	meth := NewSnowflakeRendezvousMethodDomainFronting()
	if meth.AMPCacheURL() != "" {
		t.Fatal("invalid amp cache URL")
	}
	const brokerURL = "https://snowflake-broker.torproject.net.global.prod.fastly.net/"
	if meth.BrokerURL() != brokerURL {
		t.Fatal("invalid broker URL")
	}
	const frontDomain = "cdn.sstatic.net"
	if meth.FrontDomain() != frontDomain {
		t.Fatal("invalid front domain")
	}
	if meth.Name() != "domain_fronting" {
		t.Fatal("invalid name")
	}
}

func TestSnowflakeMethodAMP(t *testing.T) {
	meth := NewSnowflakeRendezvousMethodAMP()
	const ampCacheURL = "https://cdn.ampproject.org/"
	if meth.AMPCacheURL() != ampCacheURL {
		t.Fatal("invalid amp cache URL")
	}
	const brokerURL = "https://snowflake-broker.torproject.net/"
	if meth.BrokerURL() != brokerURL {
		t.Fatal("invalid broker URL")
	}
	const frontDomain = "www.google.com"
	if meth.FrontDomain() != frontDomain {
		t.Fatal("invalid front domain")
	}
	if meth.Name() != "amp" {
		t.Fatal("invalid name")
	}
}

func TestNewSnowflakeRendezvousMethod(t *testing.T) {
	t.Run("for domain_fronted", func(t *testing.T) {
		meth, err := NewSnowflakeRendezvousMethod("domain_fronting")
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := meth.(*snowflakeRendezvousMethodDomainFronting); !ok {
			t.Fatal("unexpected method type")
		}
	})

	t.Run("for empty string", func(t *testing.T) {
		meth, err := NewSnowflakeRendezvousMethod("")
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := meth.(*snowflakeRendezvousMethodDomainFronting); !ok {
			t.Fatal("unexpected method type")
		}
	})

	t.Run("for amp", func(t *testing.T) {
		meth, err := NewSnowflakeRendezvousMethod("amp")
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := meth.(*snowflakeRendezvousMethodAMP); !ok {
			t.Fatal("unexpected method type")
		}
	})

	t.Run("for another value", func(t *testing.T) {
		meth, err := NewSnowflakeRendezvousMethod("amptani")
		if !errors.Is(err, ErrSnowflakeNoSuchRendezvousMethod) {
			t.Fatal("unexpected error", err)
		}
		if meth != nil {
			t.Fatal("unexpected method value")
		}
	})
}

func TestNewSnowflakeDialer(t *testing.T) {
	dialer := NewSnowflakeDialer()
	_, ok := dialer.RendezvousMethod.(*snowflakeRendezvousMethodDomainFronting)
	if !ok {
		t.Fatal("invalid rendezvous method type")
	}
}

func TestNewSnowflakeDialerWithRendezvousMethod(t *testing.T) {
	meth := NewSnowflakeRendezvousMethodAMP()
	dialer := NewSnowflakeDialerWithRendezvousMethod(meth)
	if meth != dialer.RendezvousMethod {
		t.Fatal("invalid rendezvous method value")
	}
}

func TestSnowflakeDialerWorks(t *testing.T) {
	// This test may sadly run for a very long time (~10s)
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	sfd := NewSnowflakeDialer()
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
		RendezvousMethod: NewSnowflakeRendezvousMethodDomainFronting(),
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
		RendezvousMethod: NewSnowflakeRendezvousMethodDomainFronting(),
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
		RendezvousMethod: NewSnowflakeRendezvousMethodDomainFronting(),
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
		RendezvousMethod: NewSnowflakeRendezvousMethodDomainFronting(),
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
	called := &atomic.Int64{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sfd := &SnowflakeDialer{
		RendezvousMethod: NewSnowflakeRendezvousMethodDomainFronting(),
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
