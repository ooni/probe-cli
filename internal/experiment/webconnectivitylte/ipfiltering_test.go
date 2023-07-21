package webconnectivitylte

import (
	"errors"
	"testing"
)

func Test_allowedToConnect(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		want     error
	}{{
		name:     "we cannot connect when there's no port",
		endpoint: "8.8.4.4",
		want:     errNotAllowedToConnect,
	}, {
		name:     "we cannot connect when address is a domain (should not happen)",
		endpoint: "dns.google:443",
		want:     errNotAllowedToConnect,
	}, {
		name:     "we cannot connect for IPv4 loopback 127.0.0.1",
		endpoint: "127.0.0.1:443",
		want:     errNotAllowedToConnect,
	}, {
		name:     "we cannot connect for IPv4 loopback 127.0.0.2",
		endpoint: "127.0.0.2:443",
		want:     errNotAllowedToConnect,
	}, {
		name:     "we can connect to 10.0.0.1 (may change in the future)",
		endpoint: "10.0.0.1:443",
		want:     nil,
	}, {
		name:     "we cannot connect for IPv6 loopback",
		endpoint: "::1",
		want:     errNotAllowedToConnect,
	}, {
		name:     "we can connect for public IPv4 address",
		endpoint: "8.8.8.8:443",
		want:     nil,
	}, {
		name:     "we can connect for public IPv6 address",
		endpoint: "[2001:4860:4860::8888]:443",
		want:     nil,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := allowedToConnect(tt.endpoint)
			if !errors.Is(err, tt.want) {
				t.Errorf("allowedToConnect() = %v, want %v", err, tt.want)
			}
		})
	}
}
