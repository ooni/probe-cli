package resolver_test

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/resolver"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/trace"
)

func TestSaverResolverFailure(t *testing.T) {
	expected := errors.New("no such host")
	saver := &trace.Saver{}
	reso := resolver.SaverResolver{
		Resolver: resolver.NewFakeResolverWithExplicitError(expected),
		Saver:    saver,
	}
	addrs, err := reso.LookupHost(context.Background(), "www.google.com")
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if addrs != nil {
		t.Fatal("expected nil address here")
	}
	ev := saver.Read()
	if len(ev) != 2 {
		t.Fatal("expected number of events")
	}
	if ev[0].Hostname != "www.google.com" {
		t.Fatal("unexpected Hostname")
	}
	if ev[0].Name != "resolve_start" {
		t.Fatal("unexpected name")
	}
	if !ev[0].Time.Before(time.Now()) {
		t.Fatal("the saved time is wrong")
	}
	if ev[1].Addresses != nil {
		t.Fatal("unexpected Addresses")
	}
	if ev[1].Duration <= 0 {
		t.Fatal("unexpected Duration")
	}
	if !errors.Is(ev[1].Err, expected) {
		t.Fatal("unexpected Err")
	}
	if ev[1].Hostname != "www.google.com" {
		t.Fatal("unexpected Hostname")
	}
	if ev[1].Name != "resolve_done" {
		t.Fatal("unexpected name")
	}
	if !ev[1].Time.After(ev[0].Time) {
		t.Fatal("the saved time is wrong")
	}
}

func TestSaverResolverSuccess(t *testing.T) {
	expected := []string{"8.8.8.8", "8.8.4.4"}
	saver := &trace.Saver{}
	reso := resolver.SaverResolver{
		Resolver: resolver.NewFakeResolverWithResult(expected),
		Saver:    saver,
	}
	addrs, err := reso.LookupHost(context.Background(), "www.google.com")
	if err != nil {
		t.Fatal("expected nil error here")
	}
	if !reflect.DeepEqual(addrs, expected) {
		t.Fatal("not the result we expected")
	}
	ev := saver.Read()
	if len(ev) != 2 {
		t.Fatal("expected number of events")
	}
	if ev[0].Hostname != "www.google.com" {
		t.Fatal("unexpected Hostname")
	}
	if ev[0].Name != "resolve_start" {
		t.Fatal("unexpected name")
	}
	if !ev[0].Time.Before(time.Now()) {
		t.Fatal("the saved time is wrong")
	}
	if !reflect.DeepEqual(ev[1].Addresses, expected) {
		t.Fatal("unexpected Addresses")
	}
	if ev[1].Duration <= 0 {
		t.Fatal("unexpected Duration")
	}
	if ev[1].Err != nil {
		t.Fatal("unexpected Err")
	}
	if ev[1].Hostname != "www.google.com" {
		t.Fatal("unexpected Hostname")
	}
	if ev[1].Name != "resolve_done" {
		t.Fatal("unexpected name")
	}
	if !ev[1].Time.After(ev[0].Time) {
		t.Fatal("the saved time is wrong")
	}
}

func TestSaverDNSTransportFailure(t *testing.T) {
	expected := errors.New("no such host")
	saver := &trace.Saver{}
	txp := resolver.SaverDNSTransport{
		DNSTransport: resolver.FakeTransport{
			Err: expected,
		},
		Saver: saver,
	}
	query := []byte("abc")
	reply, err := txp.RoundTrip(context.Background(), query)
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if reply != nil {
		t.Fatal("expected nil reply here")
	}
	ev := saver.Read()
	if len(ev) != 2 {
		t.Fatal("expected number of events")
	}
	if !bytes.Equal(ev[0].DNSQuery, query) {
		t.Fatal("unexpected DNSQuery")
	}
	if ev[0].Name != "dns_round_trip_start" {
		t.Fatal("unexpected name")
	}
	if !ev[0].Time.Before(time.Now()) {
		t.Fatal("the saved time is wrong")
	}
	if !bytes.Equal(ev[1].DNSQuery, query) {
		t.Fatal("unexpected DNSQuery")
	}
	if ev[1].DNSReply != nil {
		t.Fatal("unexpected DNSReply")
	}
	if ev[1].Duration <= 0 {
		t.Fatal("unexpected Duration")
	}
	if !errors.Is(ev[1].Err, expected) {
		t.Fatal("unexpected Err")
	}
	if ev[1].Name != "dns_round_trip_done" {
		t.Fatal("unexpected name")
	}
	if !ev[1].Time.After(ev[0].Time) {
		t.Fatal("the saved time is wrong")
	}
}

func TestSaverDNSTransportSuccess(t *testing.T) {
	expected := []byte("def")
	saver := &trace.Saver{}
	txp := resolver.SaverDNSTransport{
		DNSTransport: resolver.FakeTransport{
			Data: expected,
		},
		Saver: saver,
	}
	query := []byte("abc")
	reply, err := txp.RoundTrip(context.Background(), query)
	if err != nil {
		t.Fatal("we expected nil error here")
	}
	if !bytes.Equal(reply, expected) {
		t.Fatal("expected another reply here")
	}
	ev := saver.Read()
	if len(ev) != 2 {
		t.Fatal("expected number of events")
	}
	if !bytes.Equal(ev[0].DNSQuery, query) {
		t.Fatal("unexpected DNSQuery")
	}
	if ev[0].Name != "dns_round_trip_start" {
		t.Fatal("unexpected name")
	}
	if !ev[0].Time.Before(time.Now()) {
		t.Fatal("the saved time is wrong")
	}
	if !bytes.Equal(ev[1].DNSQuery, query) {
		t.Fatal("unexpected DNSQuery")
	}
	if !bytes.Equal(ev[1].DNSReply, expected) {
		t.Fatal("unexpected DNSReply")
	}
	if ev[1].Duration <= 0 {
		t.Fatal("unexpected Duration")
	}
	if ev[1].Err != nil {
		t.Fatal("unexpected Err")
	}
	if ev[1].Name != "dns_round_trip_done" {
		t.Fatal("unexpected name")
	}
	if !ev[1].Time.After(ev[0].Time) {
		t.Fatal("the saved time is wrong")
	}
}
