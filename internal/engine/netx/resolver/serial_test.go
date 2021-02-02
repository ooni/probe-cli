package resolver_test

import (
	"context"
	"errors"
	"net"
	"strings"
	"syscall"
	"testing"

	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/resolver"
)

func TestOONIGettingTransport(t *testing.T) {
	txp := resolver.NewDNSOverTLS(resolver.DialTLSContext, "8.8.8.8:853")
	r := resolver.NewSerialResolver(txp)
	rtx := r.Transport()
	if rtx.Network() != "dot" || rtx.Address() != "8.8.8.8:853" {
		t.Fatal("not the transport we expected")
	}
	if r.Network() != rtx.Network() {
		t.Fatal("invalid network seen from the resolver")
	}
	if r.Address() != rtx.Address() {
		t.Fatal("invalid address seen from the resolver")
	}
}

func TestOONIEncodeError(t *testing.T) {
	mocked := errors.New("mocked error")
	txp := resolver.NewDNSOverTLS(resolver.DialTLSContext, "8.8.8.8:853")
	r := resolver.SerialResolver{Encoder: resolver.FakeEncoder{Err: mocked}, Txp: txp}
	addrs, err := r.LookupHost(context.Background(), "www.gogle.com")
	if !errors.Is(err, mocked) {
		t.Fatal("not the error we expected")
	}
	if addrs != nil {
		t.Fatal("expected nil address here")
	}
}

func TestOONIRoundTripError(t *testing.T) {
	mocked := errors.New("mocked error")
	txp := resolver.FakeTransport{Err: mocked}
	r := resolver.NewSerialResolver(txp)
	addrs, err := r.LookupHost(context.Background(), "www.gogle.com")
	if !errors.Is(err, mocked) {
		t.Fatal("not the error we expected")
	}
	if addrs != nil {
		t.Fatal("expected nil address here")
	}
}

func TestOONIWithEmptyReply(t *testing.T) {
	txp := resolver.FakeTransport{Data: resolver.GenReplySuccess(t, dns.TypeA)}
	r := resolver.NewSerialResolver(txp)
	addrs, err := r.LookupHost(context.Background(), "www.gogle.com")
	if err == nil || !strings.HasSuffix(err.Error(), "no response returned") {
		t.Fatal("not the error we expected")
	}
	if addrs != nil {
		t.Fatal("expected nil address here")
	}
}

func TestOONIWithAReply(t *testing.T) {
	txp := resolver.FakeTransport{
		Data: resolver.GenReplySuccess(t, dns.TypeA, "8.8.8.8"),
	}
	r := resolver.NewSerialResolver(txp)
	addrs, err := r.LookupHost(context.Background(), "www.gogle.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(addrs) != 1 || addrs[0] != "8.8.8.8" {
		t.Fatal("not the result we expected")
	}
}

func TestOONIWithAAAAReply(t *testing.T) {
	txp := resolver.FakeTransport{
		Data: resolver.GenReplySuccess(t, dns.TypeAAAA, "::1"),
	}
	r := resolver.NewSerialResolver(txp)
	addrs, err := r.LookupHost(context.Background(), "www.gogle.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(addrs) != 1 || addrs[0] != "::1" {
		t.Fatal("not the result we expected")
	}
}

func TestOONIWithTimeout(t *testing.T) {
	txp := resolver.FakeTransport{
		Err: &net.OpError{Err: syscall.ETIMEDOUT, Op: "dial"},
	}
	r := resolver.NewSerialResolver(txp)
	addrs, err := r.LookupHost(context.Background(), "www.gogle.com")
	if !errors.Is(err, syscall.ETIMEDOUT) {
		t.Fatal("not the error we expected")
	}
	if addrs != nil {
		t.Fatal("expected nil address here")
	}
	if r.NumTimeouts.Load() <= 0 {
		t.Fatal("we didn't actually take the timeouts")
	}
}
