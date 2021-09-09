package resolver

import (
	"context"
	"errors"
	"net"
	"testing"
)

func TestDNSOverUDPDialFailure(t *testing.T) {
	mocked := errors.New("mocked error")
	const address = "9.9.9.9:53"
	txp := NewDNSOverUDP(FakeDialer{Err: mocked}, address)
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
	txp := NewDNSOverUDP(
		FakeDialer{
			Conn: &FakeConn{
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
	txp := NewDNSOverUDP(
		FakeDialer{
			Conn: &FakeConn{
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
	txp := NewDNSOverUDP(
		FakeDialer{
			Conn: &FakeConn{
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
	txp := NewDNSOverUDP(
		FakeDialer{
			Conn: &FakeConn{ReadData: make([]byte, 17)},
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
	txp := NewDNSOverUDP(&net.Dialer{}, address)
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
