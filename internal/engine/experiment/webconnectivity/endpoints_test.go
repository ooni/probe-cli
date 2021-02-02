package webconnectivity_test

import (
	"net/url"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/engine/atomicx"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/webconnectivity"
)

func TestNewEndpointPortPanicsWithInvalidScheme(t *testing.T) {
	counter := atomicx.NewInt64()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer func() {
			if recover() != nil {
				counter.Add(1)
			}
			wg.Done()
		}()
		webconnectivity.NewEndpointPort(&url.URL{Scheme: "antani"})
	}()
	wg.Wait()
	if counter.Load() != 1 {
		t.Fatal("did not panic")
	}
}

func TestNewEndpointPortPanicsWithInvalidHost(t *testing.T) {
	counter := atomicx.NewInt64()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer func() {
			if recover() != nil {
				counter.Add(1)
			}
			wg.Done()
		}()
		webconnectivity.NewEndpointPort(&url.URL{Scheme: "http", Host: "[::1"})
	}()
	wg.Wait()
	if counter.Load() != 1 {
		t.Fatal("did not panic")
	}
}

func TestNewEndpointPortCommonCase(t *testing.T) {
	type args struct {
		URL *url.URL
	}
	tests := []struct {
		name    string
		args    args
		wantOut webconnectivity.EndpointPort
	}{{
		name: "with http and no default port",
		args: args{URL: &url.URL{
			Scheme: "http",
			Host:   "www.example.com",
			Path:   "/",
		}},
		wantOut: webconnectivity.EndpointPort{
			URLGetterScheme: "tcpconnect",
			Port:            "80",
		},
	}, {
		name: "with https and no default port",
		args: args{URL: &url.URL{
			Scheme: "https",
			Host:   "www.example.com",
			Path:   "/",
		}},
		wantOut: webconnectivity.EndpointPort{
			URLGetterScheme: "tlshandshake",
			Port:            "443",
		},
	}, {
		name: "with http and custom port",
		args: args{URL: &url.URL{
			Scheme: "http",
			Host:   "www.example.com:11",
			Path:   "/",
		}},
		wantOut: webconnectivity.EndpointPort{
			URLGetterScheme: "tcpconnect",
			Port:            "11",
		},
	}, {
		name: "with https and custom port",
		args: args{URL: &url.URL{
			Scheme: "https",
			Host:   "www.example.com:11",
			Path:   "/",
		}},
		wantOut: webconnectivity.EndpointPort{
			URLGetterScheme: "tlshandshake",
			Port:            "11",
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOut := webconnectivity.NewEndpointPort(tt.args.URL)
			if diff := cmp.Diff(tt.wantOut, gotOut); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestNewEndpoints(t *testing.T) {
	type args struct {
		URL   *url.URL
		addrs []string
	}
	tests := []struct {
		name    string
		args    args
		wantOut webconnectivity.EndpointsList
	}{{
		name: "with all empty",
		args: args{
			URL: &url.URL{
				Scheme: "http",
			},
		},
		wantOut: webconnectivity.EndpointsList{},
	}, {
		name: "with some https endpoints",
		args: args{
			URL: &url.URL{
				Scheme: "https",
			},
			addrs: []string{"1.1.1.1", "8.8.8.8"},
		},
		wantOut: webconnectivity.EndpointsList{{
			URLGetterURL: "tlshandshake://1.1.1.1:443",
			String:       "1.1.1.1:443",
		}, {
			URLGetterURL: "tlshandshake://8.8.8.8:443",
			String:       "8.8.8.8:443",
		}},
	}, {
		name: "with some http endpoints",
		args: args{
			URL: &url.URL{
				Scheme: "http",
			},
			addrs: []string{"2001:4860:4860::8888", "2001:4860:4860::8844"},
		},
		wantOut: webconnectivity.EndpointsList{{
			URLGetterURL: "tcpconnect://[2001:4860:4860::8888]:80",
			String:       "[2001:4860:4860::8888]:80",
		}, {
			URLGetterURL: "tcpconnect://[2001:4860:4860::8844]:80",
			String:       "[2001:4860:4860::8844]:80",
		}},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOut := webconnectivity.NewEndpoints(tt.args.URL, tt.args.addrs)
			if diff := cmp.Diff(tt.wantOut, gotOut); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestEndpointsList_Endpoints(t *testing.T) {
	tests := []struct {
		name    string
		el      webconnectivity.EndpointsList
		wantOut []string
	}{{
		name:    "when empty",
		wantOut: []string{},
	}, {
		name: "common case",
		el: webconnectivity.EndpointsList{{
			String: "1.1.1.1:443",
		}, {
			String: "8.8.8.8:80",
		}},
		wantOut: []string{"1.1.1.1:443", "8.8.8.8:80"},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOut := tt.el.Endpoints()
			if diff := cmp.Diff(tt.wantOut, gotOut); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestEndpointsList_URLs(t *testing.T) {
	tests := []struct {
		name    string
		el      webconnectivity.EndpointsList
		wantOut []string
	}{{
		name:    "when empty",
		wantOut: []string{},
	}, {
		name: "common case",
		el: webconnectivity.EndpointsList{{
			URLGetterURL: "tlshandshake://1.1.1.1:443",
		}, {
			URLGetterURL: "tcpconnect://8.8.8.8:80",
		}},
		wantOut: []string{"tlshandshake://1.1.1.1:443", "tcpconnect://8.8.8.8:80"},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOut := tt.el.URLs()
			if diff := cmp.Diff(tt.wantOut, gotOut); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
