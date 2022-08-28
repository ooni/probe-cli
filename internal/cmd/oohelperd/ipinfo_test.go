package main

import (
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/webconnectivity"
)

func Test_newIPInfo(t *testing.T) {
	type args struct {
		creq  *ctrlRequest
		addrs []string
	}
	tests := []struct {
		name string
		args args
		want map[string]*webconnectivity.ControlIPInfo
	}{{
		name: "with empty input",
		args: args{
			creq: &webconnectivity.ControlRequest{
				HTTPRequest:        "",
				HTTPRequestHeaders: map[string][]string{},
				TCPConnect:         []string{},
			},
			addrs: []string{},
		},
		want: map[string]*webconnectivity.ControlIPInfo{},
	}, {
		name: "typical case with also bogons",
		args: args{
			creq: &webconnectivity.ControlRequest{
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
		want: map[string]*webconnectivity.ControlIPInfo{
			"10.0.0.1": {
				ASN:   0,
				Flags: webconnectivity.ControlIPInfoFlagIsBogon | webconnectivity.ControlIPInfoFlagResolvedByProbe,
			},
			"8.8.8.8": {
				ASN:   15169,
				Flags: webconnectivity.ControlIPInfoFlagResolvedByProbe | webconnectivity.ControlIPInfoFlagResolvedByTH,
			},
			"8.8.4.4": {
				ASN:   15169,
				Flags: webconnectivity.ControlIPInfoFlagResolvedByTH,
			},
		},
	}, {
		name: "with invalid endpoint",
		args: args{
			creq: &webconnectivity.ControlRequest{
				HTTPRequest:        "",
				HTTPRequestHeaders: map[string][]string{},
				TCPConnect: []string{
					"1.2.3.4",
				},
			},
			addrs: []string{},
		},
		want: map[string]*webconnectivity.ControlIPInfo{},
	}, {
		name: "with invalid IP addr",
		args: args{
			creq: &webconnectivity.ControlRequest{
				HTTPRequest:        "",
				HTTPRequestHeaders: map[string][]string{},
				TCPConnect: []string{
					"dns.google:443",
				},
			},
			addrs: []string{},
		},
		want: map[string]*webconnectivity.ControlIPInfo{},
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
		ipinfo map[string]*webconnectivity.ControlIPInfo
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
			ipinfo: map[string]*webconnectivity.ControlIPInfo{},
		},
		want: []endpointInfo{},
	}, {
		name: "with http scheme, bogons, and and no port",
		args: args{
			URL: &url.URL{
				Scheme: "http",
			},
			ipinfo: map[string]*webconnectivity.ControlIPInfo{
				"10.0.0.1": {
					ASN:   0,
					Flags: webconnectivity.ControlIPInfoFlagIsBogon | webconnectivity.ControlIPInfoFlagResolvedByProbe,
				},
				"8.8.8.8": {
					ASN:   15169,
					Flags: webconnectivity.ControlIPInfoFlagResolvedByProbe | webconnectivity.ControlIPInfoFlagResolvedByTH,
				},
				"8.8.4.4": {
					ASN:   15169,
					Flags: webconnectivity.ControlIPInfoFlagResolvedByTH,
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
			ipinfo: map[string]*webconnectivity.ControlIPInfo{
				"10.0.0.1": {
					ASN:   0,
					Flags: webconnectivity.ControlIPInfoFlagIsBogon | webconnectivity.ControlIPInfoFlagResolvedByProbe,
				},
				"8.8.8.8": {
					ASN:   15169,
					Flags: webconnectivity.ControlIPInfoFlagResolvedByProbe | webconnectivity.ControlIPInfoFlagResolvedByTH,
				},
				"8.8.4.4": {
					ASN:   15169,
					Flags: webconnectivity.ControlIPInfoFlagResolvedByTH,
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
			ipinfo: map[string]*webconnectivity.ControlIPInfo{
				"10.0.0.1": {
					ASN:   0,
					Flags: webconnectivity.ControlIPInfoFlagIsBogon | webconnectivity.ControlIPInfoFlagResolvedByProbe,
				},
				"8.8.8.8": {
					ASN:   15169,
					Flags: webconnectivity.ControlIPInfoFlagResolvedByProbe | webconnectivity.ControlIPInfoFlagResolvedByTH,
				},
				"8.8.4.4": {
					ASN:   15169,
					Flags: webconnectivity.ControlIPInfoFlagResolvedByTH,
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
			ipinfo: map[string]*webconnectivity.ControlIPInfo{
				"10.0.0.1": {
					ASN:   0,
					Flags: webconnectivity.ControlIPInfoFlagIsBogon | webconnectivity.ControlIPInfoFlagResolvedByProbe,
				},
				"8.8.8.8": {
					ASN:   15169,
					Flags: webconnectivity.ControlIPInfoFlagResolvedByProbe | webconnectivity.ControlIPInfoFlagResolvedByTH,
				},
				"8.8.4.4": {
					ASN:   15169,
					Flags: webconnectivity.ControlIPInfoFlagResolvedByTH,
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
