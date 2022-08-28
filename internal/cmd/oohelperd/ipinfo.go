package main

//
// Generates IP and endpoint information.
//

import (
	"net"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/webconnectivity"
	"github.com/ooni/probe-cli/v3/internal/engine/geolocate"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// newIPInfo creates an IP to IPInfo mapping from addresses resolved
// by the probe (inside [creq]) or the TH (inside [addrs]).
func newIPInfo(creq *ctrlRequest, addrs []string) map[string]webconnectivity.ControlIPInfo {
	discoveredby := make(map[string]int64)
	for _, epnt := range creq.TCPConnect {
		addr, _, err := net.SplitHostPort(epnt)
		if err != nil || net.ParseIP(addr) == nil {
			continue
		}
		discoveredby[addr] |= webconnectivity.ControlIPInfoFlagResolvedByProbe
	}
	for _, addr := range addrs {
		if net.ParseIP(addr) != nil {
			discoveredby[addr] |= webconnectivity.ControlIPInfoFlagResolvedByTH
		}
	}
	ipinfo := make(map[string]webconnectivity.ControlIPInfo)
	for addr, flags := range discoveredby {
		if netxlite.IsBogon(addr) { // note: we already excluded non-IP addrs above
			flags |= webconnectivity.ControlIPInfoFlagIsBogon
		}
		asn, _, _ := geolocate.LookupASN(addr) // AS0 on failure
		ipinfo[addr] = webconnectivity.ControlIPInfo{
			ASN:   int64(asn),
			Flags: flags,
		}
	}
	return ipinfo
}

// endpointInfo contains info about an endpoint to measure
type endpointInfo struct {
	// epnt is the endpoint to measure
	epnt string

	// tls indicates whether we should try using TLS
	tls bool
}

// ipInfoToEndpoints takes in input the [ipinfo] returned by newIPInfo
// and the [URL] provided by the probe to generate the list of endpoints
// to measure. When the [URL] does not contain a port, we check both
// ports 80 and 443 to have more confidence about whether IPs are valid
// for the domain. Otherwise, we just use the [URL]'s port. This function
// excludes all bogons from the returned endpoint info.
func ipInfoToEndpoints(URL *url.URL, ipinfo map[string]webconnectivity.ControlIPInfo) []endpointInfo {
	ports := []string{"80", "443"}
	if port := URL.Port(); port != "" {
		ports = []string{port} // as documented
	}
	out := []endpointInfo{}
	for addr, info := range ipinfo {
		if info.Flags&webconnectivity.ControlIPInfoFlagIsBogon != 0 {
			continue // as documented
		}
		for _, port := range ports {
			epnt := net.JoinHostPort(addr, port)
			out = append(out, endpointInfo{
				epnt: epnt,
				tls:  port == "443",
			})
		}
	}
	return out
}
