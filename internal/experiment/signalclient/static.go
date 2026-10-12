package signalclient

//
// The static addresses Signal's clients fall back to when DNS fails (ts-041 Appendix A), the
// candidates of the static_address controls. Read from libsignal v0.105.0 rust/net/src/env.rs,
// Signal-Android v8.31.1 app/static-ips.properties and Signal-Desktop 5c1a030
// build/dns-fallback.json. The StaticAddresses option replaces a host's list.
//

import (
	"fmt"
	"net"
	"strings"
)

// staticAddress is one fallback address and the client list it comes from ("libsignal",
// "android", "desktop", or "option" when set by StaticAddresses).
type staticAddress struct {
	ip     string
	source string
}

// staticTable is Appendix A: host, then the libsignal, Android and Desktop lists.
var staticTable = []struct {
	host                        string
	libsignal, android, desktop string
}{
	{"grpc.chat.signal.org",
		"76.223.66.180, 15.197.251.99, 2600:9000:a507:ab6d:575d:9d9f:64af:7a5a, 2600:9000:a61f:527c:8e4a:4b48:bbfd:c9bb",
		"", ""},
	{"chat.signal.org", "",
		"13.248.212.111, 76.223.92.165",
		"13.248.212.111, 76.223.92.165, 2600:9000:a507:ab6d:4ce3:2f58:25d7:9cbf, 2600:9000:a61f:527c:d5eb:a431:5239:3232"},
	{"cdsi.signal.org", "40.122.45.194, 2603:1030:7::1", "40.122.45.194", "40.122.45.194, 2603:1030:7::1"},
	{"svr2.signal.org",
		"20.236.21.158, 20.104.52.125, 20.9.45.98, 20.66.40.69, 20.119.62.85, 20.65.43.198, 13.84.216.212, " +
			"2603:1030:20e:33::6, 2603:1030:408:3::1d, 2603:1030:b:2a::12, 2603:1030:803:4::65, " +
			"2a01:111:f100:3000::a83e:1208, 2603:1030:c04:1e::31c, 2603:1030:f00::17",
		"20.119.62.85", ""},
	{"svrb.signal.org",
		"4.151.136.48, 20.232.191.209, 135.119.74.80, 172.200.87.186, 20.63.12.55, 20.66.41.177, 20.114.45.6, " +
			"2603:1030:20c:6::166, 2603:1030:408:6::e5, 2603:1030:7:5::22, 2a01:111:f100:4001::4625:a047, " +
			"2a01:111:f100:3000::a83e:14da, 2603:1030:c02:5::632, 2603:1030:f00:3::27",
		"", ""},
	{"storage.signal.org", "", "192.178.155.121", "192.178.142.121, 2607:f8b0:4023:80d::79"},
	{"cdn.signal.org", "",
		"18.238.49.6, 18.238.49.66, 18.238.49.90, 18.238.49.106",
		"108.139.10.15, 108.139.10.23, 108.139.10.35, 108.139.10.52, 2600:9000:234b:400:1d:4f32:50c0:93a1, " +
			"2600:9000:234b:4200:1d:4f32:50c0:93a1, 2600:9000:234b:4400:1d:4f32:50c0:93a1, " +
			"2600:9000:234b:4a00:1d:4f32:50c0:93a1, 2600:9000:234b:4c00:1d:4f32:50c0:93a1, " +
			"2600:9000:234b:8e00:1d:4f32:50c0:93a1, 2600:9000:234b:ac00:1d:4f32:50c0:93a1, " +
			"2600:9000:234b:b200:1d:4f32:50c0:93a1"},
	{"cdn2.signal.org", "", "104.18.10.47, 104.18.11.47", "104.18.10.47, 104.18.11.47, 2606:4700::6812:a2f, 2606:4700::6812:b2f"},
	{"cdn3.signal.org", "", "104.18.10.47, 104.18.11.47", "104.18.10.47, 104.18.11.47, 2606:4700::6812:a2f, 2606:4700::6812:b2f"},
	{"sfu.voip.signal.org", "", "34.117.136.13", "34.117.136.13, 2600:1901:0:e149::"},
	{"updates2.signal.org", "", "", "104.18.26.44, 104.18.27.44, 2606:4700::6812:1a2c, 2606:4700::6812:1b2c"},
}

// builtinStaticAddresses returns each host's addresses, deduplicated, each with the first list
// that has it (libsignal, then Android, then Desktop).
func builtinStaticAddresses() map[string][]staticAddress {
	out := map[string][]staticAddress{}
	for _, row := range staticTable {
		seen := map[string]bool{}
		for _, list := range []struct{ source, addrs string }{
			{"libsignal", row.libsignal}, {"android", row.android}, {"desktop", row.desktop}} {
			for _, a := range strings.Split(list.addrs, ",") {
				if a = strings.TrimSpace(a); a != "" && !seen[a] {
					seen[a] = true
					out[row.host] = append(out[row.host], staticAddress{ip: a, source: list.source})
				}
			}
		}
	}
	return out
}

// parseStaticAddresses parses the StaticAddresses option, "host=addr,addr;host=addr", into
// replacement lists.
func parseStaticAddresses(option string) (map[string][]staticAddress, error) {
	out := map[string][]staticAddress{}
	for _, entry := range strings.Split(option, ";") {
		if entry = strings.TrimSpace(entry); entry == "" {
			continue
		}
		host, addrs, ok := strings.Cut(entry, "=")
		host = strings.ToLower(strings.TrimSpace(host))
		if !ok || host == "" {
			return nil, fmt.Errorf("signal_client: StaticAddresses: invalid entry %q", entry)
		}
		var list []staticAddress
		for _, a := range strings.Split(addrs, ",") {
			a = strings.TrimSpace(a)
			if net.ParseIP(a) == nil {
				return nil, fmt.Errorf("signal_client: StaticAddresses: %q is not an IP address", a)
			}
			list = append(list, staticAddress{ip: a, source: "option"})
		}
		out[host] = list
	}
	return out, nil
}

// staticCandidates picks a target's static_address candidates: up to two IPv4 addresses and one
// IPv6 address from list, in list order, skipping tried.
func staticCandidates(list []staticAddress, tried string) []staticAddress {
	var v4, v6 []staticAddress
	for _, a := range list {
		if a.ip == tried {
			continue
		}
		if net.ParseIP(a.ip).To4() != nil {
			if len(v4) < 2 {
				v4 = append(v4, a)
			}
		} else if len(v6) < 1 {
			v6 = append(v6, a)
		}
	}
	return append(v4, v6...)
}

// firstStatic returns the first IPv4 address of list (the first address when none is IPv4).
func firstStatic(list []staticAddress) (staticAddress, bool) {
	for _, a := range list {
		if net.ParseIP(a.ip).To4() != nil {
			return a, true
		}
	}
	if len(list) > 0 {
		return list[0], true
	}
	return staticAddress{}, false
}

// inList reports whether ip is in list.
func inList(list []staticAddress, ip string) bool {
	for _, a := range list {
		if a.ip == ip {
			return true
		}
	}
	return false
}
