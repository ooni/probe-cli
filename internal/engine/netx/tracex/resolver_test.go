package tracex

import (
	"bytes"
	"context"
	"errors"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func TestSaverResolverFailure(t *testing.T) {
	expected := errors.New("no such host")
	saver := &Saver{}
	reso := saver.WrapResolver(NewFakeResolverWithExplicitError(expected))
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
	if ev[0].Value().Hostname != "www.google.com" {
		t.Fatal("unexpected Hostname")
	}
	if ev[0].Name() != "resolve_start" {
		t.Fatal("unexpected name")
	}
	if !ev[0].Value().Time.Before(time.Now()) {
		t.Fatal("the saved time is wrong")
	}
	if ev[1].Value().Addresses != nil {
		t.Fatal("unexpected Addresses")
	}
	if ev[1].Value().Duration <= 0 {
		t.Fatal("unexpected Duration")
	}
	if !errors.Is(ev[1].Value().Err, expected) {
		t.Fatal("unexpected Err")
	}
	if ev[1].Value().Hostname != "www.google.com" {
		t.Fatal("unexpected Hostname")
	}
	if ev[1].Name() != "resolve_done" {
		t.Fatal("unexpected name")
	}
	if !ev[1].Value().Time.After(ev[0].Value().Time) {
		t.Fatal("the saved time is wrong")
	}
}

func TestSaverResolverSuccess(t *testing.T) {
	expected := []string{"8.8.8.8", "8.8.4.4"}
	saver := &Saver{}
	reso := saver.WrapResolver(NewFakeResolverWithResult(expected))
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
	if ev[0].Value().Hostname != "www.google.com" {
		t.Fatal("unexpected Hostname")
	}
	if ev[0].Name() != "resolve_start" {
		t.Fatal("unexpected name")
	}
	if !ev[0].Value().Time.Before(time.Now()) {
		t.Fatal("the saved time is wrong")
	}
	if !reflect.DeepEqual(ev[1].Value().Addresses, expected) {
		t.Fatal("unexpected Addresses")
	}
	if ev[1].Value().Duration <= 0 {
		t.Fatal("unexpected Duration")
	}
	if ev[1].Value().Err != nil {
		t.Fatal("unexpected Err")
	}
	if ev[1].Value().Hostname != "www.google.com" {
		t.Fatal("unexpected Hostname")
	}
	if ev[1].Name() != "resolve_done" {
		t.Fatal("unexpected name")
	}
	if !ev[1].Value().Time.After(ev[0].Value().Time) {
		t.Fatal("the saved time is wrong")
	}
}

func TestSaverDNSTransportFailure(t *testing.T) {
	expected := errors.New("no such host")
	saver := &Saver{}
	txp := saver.WrapDNSTransport(&mocks.DNSTransport{
		MockRoundTrip: func(ctx context.Context, query model.DNSQuery) (model.DNSResponse, error) {
			return nil, expected
		},
		MockNetwork: func() string {
			return "fake"
		},
		MockAddress: func() string {
			return ""
		},
	})
	rawQuery := []byte{0xde, 0xad, 0xbe, 0xef}
	query := &mocks.DNSQuery{
		MockBytes: func() ([]byte, error) {
			return rawQuery, nil
		},
	}
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
	if !bytes.Equal(ev[0].Value().DNSQuery, rawQuery) {
		t.Fatal("unexpected DNSQuery")
	}
	if ev[0].Name() != "dns_round_trip_start" {
		t.Fatal("unexpected name")
	}
	if !ev[0].Value().Time.Before(time.Now()) {
		t.Fatal("the saved time is wrong")
	}
	if !bytes.Equal(ev[1].Value().DNSQuery, rawQuery) {
		t.Fatal("unexpected DNSQuery")
	}
	if ev[1].Value().DNSResponse != nil {
		t.Fatal("unexpected DNSReply")
	}
	if ev[1].Value().Duration <= 0 {
		t.Fatal("unexpected Duration")
	}
	if !errors.Is(ev[1].Value().Err, expected) {
		t.Fatal("unexpected Err")
	}
	if ev[1].Name() != "dns_round_trip_done" {
		t.Fatal("unexpected name")
	}
	if !ev[1].Value().Time.After(ev[0].Value().Time) {
		t.Fatal("the saved time is wrong")
	}
}

func TestSaverDNSTransportSuccess(t *testing.T) {
	expected := []byte{0xef, 0xbe, 0xad, 0xde}
	saver := &Saver{}
	response := &mocks.DNSResponse{
		MockBytes: func() []byte {
			return expected
		},
	}
	txp := saver.WrapDNSTransport(&mocks.DNSTransport{
		MockRoundTrip: func(ctx context.Context, query model.DNSQuery) (model.DNSResponse, error) {
			return response, nil
		},
		MockNetwork: func() string {
			return "fake"
		},
		MockAddress: func() string {
			return ""
		},
	})
	rawQuery := []byte{0xde, 0xad, 0xbe, 0xef}
	query := &mocks.DNSQuery{
		MockBytes: func() ([]byte, error) {
			return rawQuery, nil
		},
	}
	reply, err := txp.RoundTrip(context.Background(), query)
	if err != nil {
		t.Fatal("we expected nil error here")
	}
	if !bytes.Equal(reply.Bytes(), expected) {
		t.Fatal("expected another reply here")
	}
	ev := saver.Read()
	if len(ev) != 2 {
		t.Fatal("expected number of events")
	}
	if !bytes.Equal(ev[0].Value().DNSQuery, rawQuery) {
		t.Fatal("unexpected DNSQuery")
	}
	if ev[0].Name() != "dns_round_trip_start" {
		t.Fatal("unexpected name")
	}
	if !ev[0].Value().Time.Before(time.Now()) {
		t.Fatal("the saved time is wrong")
	}
	if !bytes.Equal(ev[1].Value().DNSQuery, rawQuery) {
		t.Fatal("unexpected DNSQuery")
	}
	if !bytes.Equal(ev[1].Value().DNSResponse, expected) {
		t.Fatal("unexpected DNSReply")
	}
	if ev[1].Value().Duration <= 0 {
		t.Fatal("unexpected Duration")
	}
	if ev[1].Value().Err != nil {
		t.Fatal("unexpected Err")
	}
	if ev[1].Name() != "dns_round_trip_done" {
		t.Fatal("unexpected name")
	}
	if !ev[1].Value().Time.After(ev[0].Value().Time) {
		t.Fatal("the saved time is wrong")
	}
}

func NewFakeResolverWithExplicitError(err error) model.Resolver {
	runtimex.PanicIfNil(err, "passed nil error")
	return &mocks.Resolver{
		MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
			return nil, err
		},
		MockNetwork: func() string {
			return "fake"
		},
		MockAddress: func() string {
			return ""
		},
		MockCloseIdleConnections: func() {
			// nothing
		},
		MockLookupHTTPS: func(ctx context.Context, domain string) (*model.HTTPSSvc, error) {
			return nil, errors.New("not implemented")
		},
		MockLookupNS: func(ctx context.Context, domain string) ([]*net.NS, error) {
			return nil, errors.New("not implemented")
		},
	}
}

func NewFakeResolverWithResult(r []string) model.Resolver {
	return &mocks.Resolver{
		MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
			return r, nil
		},
		MockNetwork: func() string {
			return "fake"
		},
		MockAddress: func() string {
			return ""
		},
		MockCloseIdleConnections: func() {
			// nothing
		},
		MockLookupHTTPS: func(ctx context.Context, domain string) (*model.HTTPSSvc, error) {
			return nil, errors.New("not implemented")
		},
		MockLookupNS: func(ctx context.Context, domain string) ([]*net.NS, error) {
			return nil, errors.New("not implemented")
		},
	}
}
