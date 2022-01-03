package resolver_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netx/handlers"
	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netx/modelx"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/resolver"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestEmitterTransportSuccess(t *testing.T) {
	ctx := context.Background()
	handler := &handlers.SavingHandler{}
	root := &modelx.MeasurementRoot{
		Beginning: time.Now(),
		Handler:   handler,
	}
	ctx = modelx.WithMeasurementRoot(ctx, root)
	txp := resolver.EmitterTransport{RoundTripper: resolver.FakeTransport{
		Data: resolver.GenReplySuccess(t, dns.TypeA, "8.8.8.8"),
	}}
	e := resolver.MiekgEncoder{}
	querydata, err := e.Encode("www.google.com", dns.TypeAAAA, true)
	if err != nil {
		t.Fatal(err)
	}
	replydata, err := txp.RoundTrip(ctx, querydata)
	if err != nil {
		t.Fatal(err)
	}
	events := handler.Read()
	if len(events) != 2 {
		t.Fatal("unexpected number of events")
	}
	if events[0].DNSQuery == nil {
		t.Fatal("missing DNSQuery field")
	}
	if !bytes.Equal(events[0].DNSQuery.Data, querydata) {
		t.Fatal("invalid query data")
	}
	if events[0].DNSQuery.DurationSinceBeginning <= 0 {
		t.Fatal("invalid duration since beginning")
	}
	if events[1].DNSReply == nil {
		t.Fatal("missing DNSReply field")
	}
	if !bytes.Equal(events[1].DNSReply.Data, replydata) {
		t.Fatal("missing reply data")
	}
	if events[1].DNSReply.DurationSinceBeginning <= 0 {
		t.Fatal("invalid duration since beginning")
	}
}

func TestEmitterTransportFailure(t *testing.T) {
	ctx := context.Background()
	handler := &handlers.SavingHandler{}
	root := &modelx.MeasurementRoot{
		Beginning: time.Now(),
		Handler:   handler,
	}
	ctx = modelx.WithMeasurementRoot(ctx, root)
	mocked := errors.New("mocked error")
	txp := resolver.EmitterTransport{RoundTripper: resolver.FakeTransport{
		Err: mocked,
	}}
	e := resolver.MiekgEncoder{}
	querydata, err := e.Encode("www.google.com", dns.TypeAAAA, true)
	if err != nil {
		t.Fatal(err)
	}
	replydata, err := txp.RoundTrip(ctx, querydata)
	if !errors.Is(err, mocked) {
		t.Fatal("not the error we expected")
	}
	if replydata != nil {
		t.Fatal("expected nil replydata")
	}
	events := handler.Read()
	if len(events) != 1 {
		t.Fatal("unexpected number of events")
	}
	if events[0].DNSQuery == nil {
		t.Fatal("missing DNSQuery field")
	}
	if !bytes.Equal(events[0].DNSQuery.Data, querydata) {
		t.Fatal("invalid query data")
	}
	if events[0].DNSQuery.DurationSinceBeginning <= 0 {
		t.Fatal("invalid duration since beginning")
	}
}

func TestEmitterResolverFailure(t *testing.T) {
	ctx := context.Background()
	handler := &handlers.SavingHandler{}
	root := &modelx.MeasurementRoot{
		Beginning: time.Now(),
		Handler:   handler,
	}
	ctx = modelx.WithMeasurementRoot(ctx, root)
	r := resolver.EmitterResolver{Resolver: resolver.NewSerialResolver(
		&resolver.DNSOverHTTPS{
			Client: &mocks.HTTPClient{
				MockDo: func(req *http.Request) (*http.Response, error) {
					return nil, io.EOF
				},
			},
			URL: "https://dns.google.com/",
		},
	)}
	replies, err := r.LookupHost(ctx, "www.google.com")
	if !errors.Is(err, io.EOF) {
		t.Fatal("not the error we expected")
	}
	if replies != nil {
		t.Fatal("expected nil replies")
	}
	events := handler.Read()
	if len(events) != 2 {
		t.Fatal("unexpected number of events")
	}
	if events[0].ResolveStart == nil {
		t.Fatal("missing ResolveStart field")
	}
	if events[0].ResolveStart.DurationSinceBeginning <= 0 {
		t.Fatal("invalid duration since beginning")
	}
	if events[0].ResolveStart.Hostname != "www.google.com" {
		t.Fatal("invalid Hostname")
	}
	if events[0].ResolveStart.TransportAddress != "https://dns.google.com/" {
		t.Fatal("invalid TransportAddress")
	}
	if events[0].ResolveStart.TransportNetwork != "doh" {
		t.Fatal("invalid TransportNetwork")
	}
	if events[1].ResolveDone == nil {
		t.Fatal("missing ResolveDone field")
	}
	if events[1].ResolveDone.DurationSinceBeginning <= 0 {
		t.Fatal("invalid duration since beginning")
	}
	if events[1].ResolveDone.Error != io.EOF {
		t.Fatal("invalid Error")
	}
	if events[1].ResolveDone.Hostname != "www.google.com" {
		t.Fatal("invalid Hostname")
	}
	if events[1].ResolveDone.TransportAddress != "https://dns.google.com/" {
		t.Fatal("invalid TransportAddress")
	}
	if events[1].ResolveDone.TransportNetwork != "doh" {
		t.Fatal("invalid TransportNetwork")
	}
}

func TestEmitterResolverSuccess(t *testing.T) {
	ctx := context.Background()
	handler := &handlers.SavingHandler{}
	root := &modelx.MeasurementRoot{
		Beginning: time.Now(),
		Handler:   handler,
	}
	ctx = modelx.WithMeasurementRoot(ctx, root)
	r := resolver.EmitterResolver{Resolver: resolver.NewFakeResolverWithResult(
		[]string{"8.8.8.8"},
	)}
	replies, err := r.LookupHost(ctx, "dns.google.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(replies) != 1 {
		t.Fatal("expected a single replies")
	}
	events := handler.Read()
	if len(events) != 2 {
		t.Fatal("unexpected number of events")
	}
	if events[1].ResolveDone == nil {
		t.Fatal("missing ResolveDone field")
	}
	if events[1].ResolveDone.Addresses[0] != "8.8.8.8" {
		t.Fatal("invalid Addresses")
	}
}
