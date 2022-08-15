package webconnectivity

//
// DNS analysis
//

import (
	"net"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/engine/geolocate"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

const (
	// AnalysisDNSBogon indicates we got any bogon reply
	AnalysisDNSBogon = 1 << iota

	// AnalysisDNSUnexpectedFailure indicates the TH could
	// resolve a domain while the probe couldn't
	AnalysisDNSUnexpectedFailure

	// AnalysisDNSUnexpectedAddrs indicates the TH resolved
	// different addresses from the probe
	AnalysisDNSUnexpectedAddrs
)

// analysisDNSToplevel is the toplevel analysis function for DNS results.
func (tk *TestKeys) analysisDNSToplevel(logger model.Logger) {
	tk.analysisDNSExperimentFailure()
	tk.analysisDNSBogon(logger)
	tk.analysisDNSUnexpectedFailure(logger)
	tk.analysisDNSUnexpectedAddrs(logger)
	tk.DNSConsistency = "consistent"
	if tk.DNSFlags != 0 {
		logger.Warnf("DNSConsistency: inconsistent")
		tk.DNSConsistency = "inconsistent"
		tk.BlockingFlags |= analysisBlockingDNS
	}
}

// analysisDNSExperimentFailure indicates whether there was any DNS
// experiment failure by inspecting all the queries.
func (tk *TestKeys) analysisDNSExperimentFailure() {
	for _, query := range tk.Queries {
		if fail := query.Failure; fail != nil {
			if query.QueryType == "AAAA" && *query.Failure == netxlite.FailureDNSNoAnswer {
				// maybe this heuristic could be further improved by checking
				// whether the TH did actually see any IPv6 address?
				continue
			}
			tk.DNSExperimentFailure = fail
			return
		}
	}
}

// analysisDNSBogon computes the AnalysisDNSBogon flag.
func (tk *TestKeys) analysisDNSBogon(logger model.Logger) {
	for _, query := range tk.Queries {
		for _, answer := range query.Answers {
			switch answer.AnswerType {
			case "A":
				if net.ParseIP(answer.IPv4) != nil && netxlite.IsBogon(answer.IPv4) {
					logger.Warnf("BOGON: %s in #%d", answer.IPv4, query.TransactionID)
					tk.DNSFlags |= AnalysisDNSBogon
					return
				}
			case "AAAA":
				if net.ParseIP(answer.IPv6) != nil && netxlite.IsBogon(answer.IPv6) {
					logger.Warnf("BOGON: %s in #%d", answer.IPv6, query.TransactionID)
					tk.DNSFlags |= AnalysisDNSBogon
					return
				}
			default:
				// nothing
			}
		}
	}
}

// analysisDNSUnexpectedFailure computes the AnalysisDNSUnexpectedFailure flags.
func (tk *TestKeys) analysisDNSUnexpectedFailure(logger model.Logger) {
	// make sure we have control before proceeding futher
	if tk.Control == nil || tk.controlRequest == nil {
		return
	}

	// obtain request and response as shortcuts
	request := tk.controlRequest
	response := tk.Control

	// obtain the domain that the TH has queried for
	URL, err := url.Parse(request.HTTPRequest)
	if err != nil {
		return // this looks like a bug
	}
	domain := URL.Hostname()

	// we obviously don't care if the domain was an IP adddress
	if net.ParseIP(domain) != nil {
		return
	}

	// we mostly care of whether the control's DNS got back
	// any IP address because this is a sign that we had
	// unexpected DNS issues locally.
	hasAddrs := len(response.DNS.Addrs) > 0
	if !hasAddrs {
		return
	}

	// therefore, any local query _for the same domain_ queried
	// by the probe that contains an error is suspicious
	for _, query := range tk.Queries {
		if domain != query.Hostname {
			continue
		}
		hasAddrs := false
	Loop:
		for _, answer := range query.Answers {
			switch answer.AnswerType {
			case "A", "AAA":
				hasAddrs = true
				break Loop
			}
		}
		if hasAddrs {
			// if the lookup returned any IP address, we are
			// not dealing with unexpected failures
			continue
		}
		if query.Failure == nil {
			// we expect to see a failure if we don't see
			// answers, so this seems a bug
			continue
		}
		if query.QueryType == "AAAA" && *query.Failure == netxlite.FailureDNSNoAnswer {
			// maybe this heuristic could be further improved by checking
			// whether the TH did actually see any IPv6 address?
			continue
		}
		logger.Warnf("DNS: unexpected failure %s in #%d", *query.Failure, query.TransactionID)
		tk.DNSFlags |= AnalysisDNSUnexpectedFailure
		return
	}
}

// analysisDNSUnexpectedAddrs computes the AnalysisDNSUnexpectedAddrs flags.
func (tk *TestKeys) analysisDNSUnexpectedAddrs(logger model.Logger) {
	// if the list of addresses for which we could not perform a TLS handshake is
	// empty, there's no need to compare with the TH, since we can use the results
	// of the TLS handshake alone to say that all addresses were correct.
	addrsWithoutTLSHandshake := tk.findAddrsWithoutTLSHandshake()
	if len(addrsWithoutTLSHandshake) <= 0 {
		return
	}
	logger.Warnf("DNS: addrs without TLS handshake: %+v", addrsWithoutTLSHandshake)

	// make sure we have control before proceeding futher
	if tk.Control == nil || tk.controlRequest == nil {
		return
	}

	// obtain request and response as shortcuts
	request := tk.controlRequest
	response := tk.Control

	// obtain the domain that the TH has queried for
	URL, err := url.Parse(request.HTTPRequest)
	if err != nil {
		return // this looks like a bug
	}
	domain := URL.Hostname()

	// we obviously don't care if the domain was an IP adddress
	if net.ParseIP(domain) != nil {
		return
	}

	// we mostly care of whether the control's DNS got back
	// any IP address because this is a sign that we had
	// unexpected DNS issues locally.
	thAddrs := response.DNS.Addrs
	if len(thAddrs) <= 0 {
		return
	}

	// gather all the IP addresses queried by the probe
	// for the same domain for which the TH queried.
	var probeAddrs []string
	for _, query := range tk.Queries {
		if domain != query.Hostname {
			continue
		}
		for _, answer := range query.Answers {
			switch answer.AnswerType {
			case "A":
				probeAddrs = append(probeAddrs, answer.IPv4)
			case "AAAA":
				probeAddrs = append(probeAddrs, answer.IPv6)
			}
		}
	}

	// if the probe has not collected any addr for the same domain, it's
	// definitely suspicious and counts as a difference
	if len(probeAddrs) <= 0 {
		logger.Warnf("DNS: no IP address resolved by the probe")
		tk.DNSFlags |= AnalysisDNSUnexpectedAddrs
		return
	}

	// if there are no different addresses between the probe and the TH then
	// our job here is done and we can just stop searching
	differentAddrs := tk.analysisDNSDiffAddrs(probeAddrs, thAddrs)
	if len(differentAddrs) <= 0 {
		return
	}

	// if the different addrs have the same ASN of addrs resolved by
	// the TH, then we say everything is still fine.
	differentASNS := tk.analysisDNSDiffASN(differentAddrs, thAddrs)
	if len(differentASNS) <= 0 {
		return
	}

	// otherwise, conclude we have unexpected probe addrs
	logger.Warnf("DNS: differentAddrs: %+v, differentASNs: %+v", differentAddrs, differentASNS)
	tk.DNSFlags |= AnalysisDNSUnexpectedAddrs
}

// analysisDNSDiffAddrs returns all the IP addresses that are
// resolved by the probe but not by the test helper.
func (tk *TestKeys) analysisDNSDiffAddrs(probeAddrs, thAddrs []string) (diff []string) {
	const (
		inProbe = 1 << iota
		inTH
	)
	mapping := make(map[string]int)
	for _, addr := range probeAddrs {
		mapping[addr] |= inProbe
	}
	for _, addr := range thAddrs {
		mapping[addr] = inTH
	}
	for addr, where := range mapping {
		if where&inTH == 0 {
			diff = append(diff, addr)
		}
	}
	return
}

// analysisDNSDiffASN returns whether there are IP addresses in the probe's
// list with different ASNs from the ones in the TH's list.
func (tk *TestKeys) analysisDNSDiffASN(probeAddrs, thAddrs []string) (asns []uint) {
	const (
		inProbe = 1 << iota
		inTH
	)
	mapping := make(map[uint]int)
	for _, addr := range probeAddrs {
		asn, _, _ := geolocate.LookupASN(addr)
		mapping[asn] |= inProbe // including the zero ASN that means unknown
	}
	for _, addr := range thAddrs {
		asn, _, _ := geolocate.LookupASN(addr)
		mapping[asn] |= inTH // including the zero ASN that means unknown
	}
	for asn, where := range mapping {
		if where&inTH == 0 {
			asns = append(asns, asn)
		}
	}
	return
}

// findAddrsWithoutTLSHandshake computes the list of probe discovered addresses
// for which we couldn't successfully perform a TLS handshake.
func (tk *TestKeys) findAddrsWithoutTLSHandshake() (output []string) {
	const (
		resolved = 1 << iota
		handshakeOK
	)
	mapping := make(map[string]int)

	// gather all the addrs resolved by the probe
	for _, query := range tk.Queries {
		for _, answer := range query.Answers {
			switch answer.AnswerType {
			case "A":
				mapping[answer.IPv4] |= resolved
			case "AAAA":
				mapping[answer.IPv6] |= resolved
			}
		}
	}

	// gather all the addrs with successful handshake
	for _, thx := range tk.TLSHandshakes {
		addr, _, err := net.SplitHostPort(thx.Address)
		if err != nil {
			continue // looks like a bug
		}
		if thx.Failure != nil {
			continue // this handshake failed
		}
		mapping[addr] |= handshakeOK
	}

	// compute the list of addresses without the handshakeOK flag
	for addr, flags := range mapping {
		if flags&handshakeOK == 0 {
			output = append(output, addr)
		}
	}
	return
}
