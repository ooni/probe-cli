package main

import (
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func Test_newIPInfo(t *testing.T) {
	type args struct {
		creq  *ctrlRequest
		addrs []string
	}
	tests := []struct {
		name string
		args args
		want map[string]*model.THIPInfo
	}{{
		name: "with empty input",
		args: args{
			creq: &model.THRequest{
				HTTPRequest:        "",
				HTTPRequestHeaders: map[string][]string{},
				TCPConnect:         []string{},
			},
			addrs: []string{},
		},
		want: map[string]*model.THIPInfo{},
	}, {
		name: "typical case with also bogons",
		args: args{
			creq: &model.THRequest{
				HTTPRequest:        "",
				HTTPRequestHeaders: map[string][]string{},
				TCPConnect: []string{
					"10.0.0.1:443",
					"8.8.8.8:443",
				},
			},
			addrs: []string{
				"8.8.8.8",
				"8.8.4.4",
			},
		},
		want: map[string]*model.THIPInfo{
			"10.0.0.1": {
				ASN:   0,
				Flags: model.THIPInfoFlagIsBogon | model.THIPInfoFlagResolvedByProbe,
			},
			"8.8.8.8": {
				ASN:   15169,
				Flags: model.THIPInfoFlagResolvedByProbe | model.THIPInfoFlagResolvedByTH,
			},
			"8.8.4.4": {
				ASN:   15169,
				Flags: model.THIPInfoFlagResolvedByTH,
			},
		},
	}, {
		name: "with invalid endpoint",
		args: args{
			creq: &model.THRequest{
				HTTPRequest:        "",
				HTTPRequestHeaders: map[string][]string{},
				TCPConnect: []string{
					"1.2.3.4",
				},
			},
			addrs: []string{},
		},
		want: map[string]*model.THIPInfo{},
	}, {
		name: "with invalid IP addr",
		args: args{
			creq: &model.THRequest{
				HTTPRequest:        "",
				HTTPRequestHeaders: map[string][]string{},
				TCPConnect: []string{
					"dns.google:443",
				},
			},
			addrs: []string{},
		},
		want: map[string]*model.THIPInfo{},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newIPInfo(tt.args.creq, tt.args.addrs)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func Test_ipInfoToEndpoints(t *testing.T) {
	type args struct {
		URL    *url.URL
		ipinfo map[string]*model.THIPInfo
	}
	tests := []struct {
		name string
		args args
		want []endpointInfo
	}{{
		name: "with nil map and empty URL",
		args: args{
			URL:    &url.URL{},
			ipinfo: nil,
		},
		want: []endpointInfo{},
	}, {
		name: "with empty map and empty URL",
		args: args{
			URL:    &url.URL{},
			ipinfo: map[string]*model.THIPInfo{},
		},
		want: []endpointInfo{},
	}, {
		name: "with http scheme, bogons, and and no port",
		args: args{
			URL: &url.URL{
				Scheme: "http",
			},
			ipinfo: map[string]*model.THIPInfo{
				"10.0.0.1": {
					ASN:   0,
					Flags: model.THIPInfoFlagIsBogon | model.THIPInfoFlagResolvedByProbe,
				},
				"8.8.8.8": {
					ASN:   15169,
					Flags: model.THIPInfoFlagResolvedByProbe | model.THIPInfoFlagResolvedByTH,
				},
				"8.8.4.4": {
					ASN:   15169,
					Flags: model.THIPInfoFlagResolvedByTH,
				},
			},
		},
		want: []endpointInfo{{
			Addr: "8.8.4.4",
			Epnt: "8.8.4.4:443",
			TLS:  true,
		}, {
			Addr: "8.8.4.4",
			Epnt: "8.8.4.4:80",
			TLS:  false,
		}, {
			Addr: "8.8.8.8",
			Epnt: "8.8.8.8:443",
			TLS:  true,
		}, {
			Addr: "8.8.8.8",
			Epnt: "8.8.8.8:80",
			TLS:  false,
		}},
	}, {
		name: "with bogons and explicit port",
		args: args{
			URL: &url.URL{
				Host: "dns.google:5432",
			},
			ipinfo: map[string]*model.THIPInfo{
				"10.0.0.1": {
					ASN:   0,
					Flags: model.THIPInfoFlagIsBogon | model.THIPInfoFlagResolvedByProbe,
				},
				"8.8.8.8": {
					ASN:   15169,
					Flags: model.THIPInfoFlagResolvedByProbe | model.THIPInfoFlagResolvedByTH,
				},
				"8.8.4.4": {
					ASN:   15169,
					Flags: model.THIPInfoFlagResolvedByTH,
				},
			},
		},
		want: []endpointInfo{{
			Addr: "8.8.4.4",
			Epnt: "8.8.4.4:5432",
			TLS:  false,
		}, {
			Addr: "8.8.8.8",
			Epnt: "8.8.8.8:5432",
			TLS:  false,
		}},
	}, {
		name: "with addresses and some bogons, no port, and unknown scheme",
		args: args{
			URL: &url.URL{},
			ipinfo: map[string]*model.THIPInfo{
				"10.0.0.1": {
					ASN:   0,
					Flags: model.THIPInfoFlagIsBogon | model.THIPInfoFlagResolvedByProbe,
				},
				"8.8.8.8": {
					ASN:   15169,
					Flags: model.THIPInfoFlagResolvedByProbe | model.THIPInfoFlagResolvedByTH,
				},
				"8.8.4.4": {
					ASN:   15169,
					Flags: model.THIPInfoFlagResolvedByTH,
				},
			},
		},
		want: []endpointInfo{},
	}, {
		name: "with addresses and some bogons, no port, and https scheme",
		args: args{
			URL: &url.URL{
				Scheme: "https",
			},
			ipinfo: map[string]*model.THIPInfo{
				"10.0.0.1": {
					ASN:   0,
					Flags: model.THIPInfoFlagIsBogon | model.THIPInfoFlagResolvedByProbe,
				},
				"8.8.8.8": {
					ASN:   15169,
					Flags: model.THIPInfoFlagResolvedByProbe | model.THIPInfoFlagResolvedByTH,
				},
				"8.8.4.4": {
					ASN:   15169,
					Flags: model.THIPInfoFlagResolvedByTH,
				},
			},
		},
		want: []endpointInfo{{
			Addr: "8.8.4.4",
			Epnt: "8.8.4.4:443",
			TLS:  true,
		}, {
			Addr: "8.8.8.8",
			Epnt: "8.8.8.8:443",
			TLS:  true,
		}},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ipInfoToEndpoints(tt.args.URL, tt.args.ipinfo)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
