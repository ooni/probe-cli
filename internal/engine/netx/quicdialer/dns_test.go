package quicdialer_test

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"strconv"
	"strings"
	"testing"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/quicdialer"
)

type MockableResolver struct {
	Addresses []string
	Err       error
}

func (r MockableResolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	return r.Addresses, r.Err
}

func TestDNSDialerSuccess(t *testing.T) {
	tlsConf := &tls.Config{NextProtos: []string{"h3-29"}}
	dialer := quicdialer.DNSDialer{
		Resolver: new(net.Resolver), Dialer: quicdialer.SystemDialer{}}
	sess, err := dialer.DialContext(
		context.Background(), "udp", "www.google.com:443",
		tlsConf, &quic.Config{})
	if err != nil {
		t.Fatal("unexpected error", err)
	}
	if sess == nil {
		t.Fatal("non nil sess expected")
	}
}

func TestDNSDialerNoPort(t *testing.T) {
	tlsConf := &tls.Config{NextProtos: []string{"h3-29"}}
	dialer := quicdialer.DNSDialer{
		Resolver: new(net.Resolver), Dialer: quicdialer.SystemDialer{}}
	sess, err := dialer.DialContext(
		context.Background(), "udp", "www.google.com",
		tlsConf, &quic.Config{})
	if err == nil {
		t.Fatal("expected an error here")
	}
	if sess != nil {
		t.Fatal("expected a nil sess here")
	}
	if err.Error() != "address www.google.com: missing port in address" {
		t.Fatal("not the error we expected")
	}
}

func TestDNSDialerLookupHostAddress(t *testing.T) {
	dialer := quicdialer.DNSDialer{Resolver: MockableResolver{
		Err: errors.New("mocked error"),
	}}
	addrs, err := dialer.LookupHost(context.Background(), "1.1.1.1")
	if err != nil {
		t.Fatal(err)
	}
	if len(addrs) != 1 || addrs[0] != "1.1.1.1" {
		t.Fatal("not the result we expected")
	}
}

func TestDNSDialerLookupHostFailure(t *testing.T) {
	tlsConf := &tls.Config{NextProtos: []string{"h3-29"}}
	expected := errors.New("mocked error")
	dialer := quicdialer.DNSDialer{Resolver: MockableResolver{
		Err: expected,
	}}
	sess, err := dialer.DialContext(
		context.Background(), "udp", "dns.google.com:853",
		tlsConf, &quic.Config{})
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if sess != nil {
		t.Fatal("expected nil sess")
	}
}

func TestDNSDialerInvalidPort(t *testing.T) {
	tlsConf := &tls.Config{NextProtos: []string{"h3-29"}}
	dialer := quicdialer.DNSDialer{
		Resolver: new(net.Resolver), Dialer: quicdialer.SystemDialer{}}
	sess, err := dialer.DialContext(
		context.Background(), "udp", "www.google.com:0",
		tlsConf, &quic.Config{})
	if err == nil {
		t.Fatal("expected an error here")
	}
	if sess != nil {
		t.Fatal("expected nil sess")
	}
	if !strings.Contains(err.Error(), "sendto: invalid argument") &&
		!strings.HasSuffix(err.Error(), "sendto: can't assign requested address") {
		t.Fatal("not the error we expected", err.Error())
	}
}

func TestDNSDialerInvalidPortSyntax(t *testing.T) {
	tlsConf := &tls.Config{NextProtos: []string{"h3-29"}}
	dialer := quicdialer.DNSDialer{
		Resolver: new(net.Resolver), Dialer: quicdialer.SystemDialer{}}
	sess, err := dialer.DialContext(
		context.Background(), "udp", "www.google.com:port",
		tlsConf, &quic.Config{})
	if err == nil {
		t.Fatal("expected an error here")
	}
	if sess != nil {
		t.Fatal("expected nil sess")
	}
	if !errors.Is(err, strconv.ErrSyntax) {
		t.Fatal("not the error we expected")
	}
}

func TestDNSDialerDialEarlyFails(t *testing.T) {
	tlsConf := &tls.Config{NextProtos: []string{"h3-29"}}
	expected := errors.New("mocked DialEarly error")
	dialer := quicdialer.DNSDialer{
		Resolver: new(net.Resolver), Dialer: MockDialer{Err: expected}}
	sess, err := dialer.DialContext(
		context.Background(), "udp", "www.google.com:443",
		tlsConf, &quic.Config{})
	if err == nil {
		t.Fatal("expected an error here")
	}
	if sess != nil {
		t.Fatal("expected nil sess")
	}
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
}
