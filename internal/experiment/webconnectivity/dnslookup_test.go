package webconnectivity_test

import (
	"context"
	"net"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/experiment/webconnectivity"
)

func TestDNSLookup(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	config := webconnectivity.DNSLookupConfig{
		Session: newsession(t, true),
		URL:     &url.URL{Host: "dns.google"},
	}
	out := webconnectivity.DNSLookup(context.Background(), config)
	if out.Failure != nil {
		t.Fatal(*out.Failure)
	}
	if len(out.Addrs) < 1 {
		t.Fatal("no addresses?!")
	}
	for addr, asn := range out.Addrs {
		if net.ParseIP(addr) == nil {
			t.Fatal("invalid addr")
		}
		if asn != 15169 {
			t.Fatal("invalid asn")
		}
	}
	if len(out.TestKeys.NetworkEvents) < 1 {
		t.Fatal("no network events?!")
	}
	if len(out.TestKeys.Queries) < 1 {
		t.Fatal("no queries?!")
	}
}

func TestDNSLookupResult_Addresses(t *testing.T) {
	type fields struct {
		Addrs    map[string]int64
		Failure  *string
		TestKeys urlgetter.TestKeys
	}
	tests := []struct {
		name    string
		fields  fields
		wantOut []string
	}{{
		name:    "with no entries",
		fields:  fields{},
		wantOut: []string{},
	}, {
		name: "with some entries",
		fields: fields{
			Addrs: map[string]int64{"1.1.1.1": 1, "2001:4860:4860::8844": 2},
		},
		wantOut: []string{"1.1.1.1", "2001:4860:4860::8844"},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := webconnectivity.DNSLookupResult{
				Addrs:    tt.fields.Addrs,
				Failure:  tt.fields.Failure,
				TestKeys: tt.fields.TestKeys,
			}
			gotOut := r.Addresses()
			if diff := cmp.Diff(tt.wantOut, gotOut); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
