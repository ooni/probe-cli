package webconnectivity

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_dnsMergeEntries(t *testing.T) {
	tests := []struct {
		name        string
		systemAddrs []string
		udpAddrs    []string
		httpsAddrs  []string
		want        map[string]int64
	}{{
		name: "we skip localhost IP addresses",
		systemAddrs: []string{
			"8.8.8.8", "127.0.0.1",
		},
		udpAddrs: []string{
			"127.0.0.2", "8.8.4.4", "::1", "8.8.8.8",
		},
		httpsAddrs: []string{
			"8.8.8.8", "::1", "2001:4860:4860::8888",
		},
		want: map[string]int64{
			"8.8.8.8":              DNSAddrFlagSystemResolver | DNSAddrFlagUDP | DNSAddrFlagHTTPS,
			"8.8.4.4":              DNSAddrFlagUDP,
			"2001:4860:4860::8888": DNSAddrFlagHTTPS,
		},
	}, {
		name: "we skip non-ip-addr entries (should not happen but just in case...)",
		systemAddrs: []string{
			"dns.google", "8.8.8.8",
		},
		udpAddrs: []string{
			"dns.google", "8.8.4.4", "8.8.8.8",
		},
		httpsAddrs: []string{
			"8.8.8.8", "2001:4860:4860::8888",
		},
		want: map[string]int64{
			"8.8.8.8":              DNSAddrFlagSystemResolver | DNSAddrFlagUDP | DNSAddrFlagHTTPS,
			"8.8.4.4":              DNSAddrFlagUDP,
			"2001:4860:4860::8888": DNSAddrFlagHTTPS,
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dnsMergeEntries(tt.systemAddrs, tt.udpAddrs, tt.httpsAddrs)
			gotmap := map[string]int64{}
			for _, entry := range got {
				if _, found := gotmap[entry.Addr]; found {
					t.Fatal("duplicate keys in result")
				}
				gotmap[entry.Addr] = entry.Flags
			}
			if diff := cmp.Diff(tt.want, gotmap); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
