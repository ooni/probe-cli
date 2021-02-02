package resolver_test

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/resolver"
)

func TestDNSOverUDPDialFailure(t *testing.T) {
	mocked := errors.New("mocked error")
	const address = "9.9.9.9:53"
	txp := resolver.NewDNSOverUDP(resolver.FakeDialer{Err: mocked}, address)
	data, err := txp.RoundTrip(context.Background(), nil)
	if !errors.Is(err, mocked) {
		t.Fatal("not the error we expected")
	}
	if data != nil {
		t.Fatal("expected no response here")
	}
}

func TestDNSOverUDPSetDeadlineError(t *testing.T) {
	mocked := errors.New("mocked error")
	txp := resolver.NewDNSOverUDP(
		resolver.FakeDialer{
			Conn: &resolver.FakeConn{
				SetDeadlineError: mocked,
			},
		}, "9.9.9.9:53",
	)
	data, err := txp.RoundTrip(context.Background(), nil)
	if !errors.Is(err, mocked) {
		t.Fatal("not the error we expected")
	}
	if data != nil {
		t.Fatal("expected no response here")
	}
}

func TestDNSOverUDPWriteFailure(t *testing.T) {
	mocked := errors.New("mocked error")
	txp := resolver.NewDNSOverUDP(
		resolver.FakeDialer{
			Conn: &resolver.FakeConn{
				WriteError: mocked,
			},
		}, "9.9.9.9:53",
	)
	data, err := txp.RoundTrip(context.Background(), nil)
	if !errors.Is(err, mocked) {
		t.Fatal("not the error we expected")
	}
	if data != nil {
		t.Fatal("expected no response here")
	}
}

func TestDNSOverUDPReadFailure(t *testing.T) {
	mocked := errors.New("mocked error")
	txp := resolver.NewDNSOverUDP(
		resolver.FakeDialer{
			Conn: &resolver.FakeConn{
				ReadError: mocked,
			},
		}, "9.9.9.9:53",
	)
	data, err := txp.RoundTrip(context.Background(), nil)
	if !errors.Is(err, mocked) {
		t.Fatal("not the error we expected")
	}
	if data != nil {
		t.Fatal("expected no response here")
	}
}

func TestDNSOverUDPReadSuccess(t *testing.T) {
	const expected = 17
	txp := resolver.NewDNSOverUDP(
		resolver.FakeDialer{
			Conn: &resolver.FakeConn{ReadData: make([]byte, 17)},
		}, "9.9.9.9:53",
	)
	data, err := txp.RoundTrip(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != expected {
		t.Fatal("expected non nil data")
	}
}

func TestDNSOverUDPTransportOK(t *testing.T) {
	const address = "9.9.9.9:53"
	txp := resolver.NewDNSOverUDP(&net.Dialer{}, address)
	if txp.RequiresPadding() != false {
		t.Fatal("invalid RequiresPadding")
	}
	if txp.Network() != "udp" {
		t.Fatal("invalid Network")
	}
	if txp.Address() != address {
		t.Fatal("invalid Address")
	}
}
