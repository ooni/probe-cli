package dialer

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/mockablex"
)

func TestDNSDialerNoPort(t *testing.T) {
	dialer := &dnsDialer{Dialer: new(net.Dialer), Resolver: new(net.Resolver)}
	conn, err := dialer.DialContext(context.Background(), "tcp", "antani.ooni.nu")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if conn != nil {
		t.Fatal("expected a nil conn here")
	}
}

func TestDNSDialerLookupHostAddress(t *testing.T) {
	dialer := &dnsDialer{Dialer: new(net.Dialer), Resolver: MockableResolver{
		Err: errors.New("mocked error"),
	}}
	addrs, err := dialer.lookupHost(context.Background(), "1.1.1.1")
	if err != nil {
		t.Fatal(err)
	}
	if len(addrs) != 1 || addrs[0] != "1.1.1.1" {
		t.Fatal("not the result we expected")
	}
}

func TestDNSDialerLookupHostFailure(t *testing.T) {
	expected := errors.New("mocked error")
	dialer := &dnsDialer{Dialer: new(net.Dialer), Resolver: MockableResolver{
		Err: expected,
	}}
	conn, err := dialer.DialContext(context.Background(), "tcp", "dns.google.com:853")
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if conn != nil {
		t.Fatal("expected nil conn")
	}
}

type MockableResolver struct {
	Addresses []string
	Err       error
}

func (r MockableResolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	return r.Addresses, r.Err
}

func TestDNSDialerDialForSingleIPFails(t *testing.T) {
	dialer := &dnsDialer{Dialer: mockablex.Dialer{
		MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
			return nil, io.EOF
		},
	}, Resolver: new(net.Resolver)}
	conn, err := dialer.DialContext(context.Background(), "tcp", "1.1.1.1:853")
	if !errors.Is(err, io.EOF) {
		t.Fatal("not the error we expected")
	}
	if conn != nil {
		t.Fatal("expected nil conn")
	}
}

func TestDNSDialerDialForManyIPFails(t *testing.T) {
	dialer := &dnsDialer{
		Dialer: mockablex.Dialer{
			MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
				return nil, io.EOF
			},
		}, Resolver: MockableResolver{
			Addresses: []string{"1.1.1.1", "8.8.8.8"},
		}}
	conn, err := dialer.DialContext(context.Background(), "tcp", "dot.dns:853")
	if !errors.Is(err, io.EOF) {
		t.Fatal("not the error we expected")
	}
	if conn != nil {
		t.Fatal("expected nil conn")
	}
}

func TestDNSDialerDialForManyIPSuccess(t *testing.T) {
	dialer := &dnsDialer{Dialer: mockablex.Dialer{
		MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
			return &mockablex.Conn{
				MockClose: func() error {
					return nil
				},
			}, nil
		},
	}, Resolver: MockableResolver{
		Addresses: []string{"1.1.1.1", "8.8.8.8"},
	}}
	conn, err := dialer.DialContext(context.Background(), "tcp", "dot.dns:853")
	if err != nil {
		t.Fatal("expected nil error here")
	}
	if conn == nil {
		t.Fatal("expected non-nil conn")
	}
	conn.Close()
}
