package resolver

import (
	"context"
	"errors"
	"net"
	"testing"
)

func TestDNSOverTCPTransportQueryTooLarge(t *testing.T) {
	const address = "9.9.9.9:53"
	txp := NewDNSOverTCP(new(net.Dialer).DialContext, address)
	reply, err := txp.RoundTrip(context.Background(), make([]byte, 1<<18))
	if err == nil {
		t.Fatal("expected an error here")
	}
	if reply != nil {
		t.Fatal("expected nil reply here")
	}
}

func TestDNSOverTCPTransportDialFailure(t *testing.T) {
	const address = "9.9.9.9:53"
	mocked := errors.New("mocked error")
	fakedialer := FakeDialer{Err: mocked}
	txp := NewDNSOverTCP(fakedialer.DialContext, address)
	reply, err := txp.RoundTrip(context.Background(), make([]byte, 1<<11))
	if !errors.Is(err, mocked) {
		t.Fatal("not the error we expected")
	}
	if reply != nil {
		t.Fatal("expected nil reply here")
	}
}

func TestDNSOverTCPTransportSetDealineFailure(t *testing.T) {
	const address = "9.9.9.9:53"
	mocked := errors.New("mocked error")
	fakedialer := FakeDialer{Conn: &FakeConn{
		SetDeadlineError: mocked,
	}}
	txp := NewDNSOverTCP(fakedialer.DialContext, address)
	reply, err := txp.RoundTrip(context.Background(), make([]byte, 1<<11))
	if !errors.Is(err, mocked) {
		t.Fatal("not the error we expected")
	}
	if reply != nil {
		t.Fatal("expected nil reply here")
	}
}

func TestDNSOverTCPTransportWriteFailure(t *testing.T) {
	const address = "9.9.9.9:53"
	mocked := errors.New("mocked error")
	fakedialer := FakeDialer{Conn: &FakeConn{
		WriteError: mocked,
	}}
	txp := NewDNSOverTCP(fakedialer.DialContext, address)
	reply, err := txp.RoundTrip(context.Background(), make([]byte, 1<<11))
	if !errors.Is(err, mocked) {
		t.Fatal("not the error we expected")
	}
	if reply != nil {
		t.Fatal("expected nil reply here")
	}
}

func TestDNSOverTCPTransportReadFailure(t *testing.T) {
	const address = "9.9.9.9:53"
	mocked := errors.New("mocked error")
	fakedialer := FakeDialer{Conn: &FakeConn{
		ReadError: mocked,
	}}
	txp := NewDNSOverTCP(fakedialer.DialContext, address)
	reply, err := txp.RoundTrip(context.Background(), make([]byte, 1<<11))
	if !errors.Is(err, mocked) {
		t.Fatal("not the error we expected")
	}
	if reply != nil {
		t.Fatal("expected nil reply here")
	}
}

func TestDNSOverTCPTransportSecondReadFailure(t *testing.T) {
	const address = "9.9.9.9:53"
	mocked := errors.New("mocked error")
	fakedialer := FakeDialer{Conn: &FakeConn{
		ReadError: mocked,
		ReadData:  []byte{byte(0), byte(2)},
	}}
	txp := NewDNSOverTCP(fakedialer.DialContext, address)
	reply, err := txp.RoundTrip(context.Background(), make([]byte, 1<<11))
	if !errors.Is(err, mocked) {
		t.Fatal("not the error we expected")
	}
	if reply != nil {
		t.Fatal("expected nil reply here")
	}
}

func TestDNSOverTCPTransportAllGood(t *testing.T) {
	const address = "9.9.9.9:53"
	mocked := errors.New("mocked error")
	fakedialer := FakeDialer{Conn: &FakeConn{
		ReadError: mocked,
		ReadData:  []byte{byte(0), byte(1), byte(1)},
	}}
	txp := NewDNSOverTCP(fakedialer.DialContext, address)
	reply, err := txp.RoundTrip(context.Background(), make([]byte, 1<<11))
	if err != nil {
		t.Fatal(err)
	}
	if len(reply) != 1 || reply[0] != 1 {
		t.Fatal("not the response we expected")
	}
}

func TestDNSOverTCPTransportOK(t *testing.T) {
	const address = "9.9.9.9:53"
	txp := NewDNSOverTCP(new(net.Dialer).DialContext, address)
	if txp.RequiresPadding() != false {
		t.Fatal("invalid RequiresPadding")
	}
	if txp.Network() != "tcp" {
		t.Fatal("invalid Network")
	}
	if txp.Address() != address {
		t.Fatal("invalid Address")
	}
}

func TestDNSOverTLSTransportOK(t *testing.T) {
	const address = "9.9.9.9:853"
	txp := NewDNSOverTLS(DialTLSContext, address)
	if txp.RequiresPadding() != true {
		t.Fatal("invalid RequiresPadding")
	}
	if txp.Network() != "dot" {
		t.Fatal("invalid Network")
	}
	if txp.Address() != address {
		t.Fatal("invalid Address")
	}
}
