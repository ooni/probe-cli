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
//
// The goals of this function are the following:
//
// 1. Set the legacy .DNSExperimentFailure field to the failure value of the
// first DNS query that failed among the ones we actually tried. Because we
// have multiple queries, unfortunately we are forced to pick one error among
// possibly many to assign to this field. This is why I consider it legacy.
//
// 2. Compute the XDNSFlags value.
//
// From the XDNSFlags value, we determine, in turn DNSConsistency and
// XBlockingFlags according to the following decision table:
//
//     +-----------+----------------+---------------------+
//     | XDNSFlags | DNSConsistency | XBlockingFlags      |
//     +-----------+----------------+---------------------+
//     | 0         | "consistent"   | no change           |
//     +-----------+----------------+---------------------+
//     | nonzero   | "inconsistent" | set FlagDNSBlocking |
//     +-----------+----------------+---------------------+
//
// We explain how XDNSFlags is determined in the documentation of
// the functions that this function calls to do its job.
func (tk *TestKeys) analysisDNSToplevel(logger model.Logger) {
	tk.analysisDNSExperimentFailure()
	tk.analysisDNSBogon(logger)
	tk.analysisDNSUnexpectedFailure(logger)
	tk.analysisDNSUnexpectedAddrs(logger)
	if tk.XDNSFlags != 0 {
		logger.Warn("DNSConsistency: inconsistent")
		tk.DNSConsistency = "inconsistent"
		tk.XBlockingFlags |= analysisFlagDNSBlocking
	} else {
		logger.Info("DNSConsistency: consistent")
		tk.DNSConsistency = "consistent"
	}
}

// analysisDNSExperimentFailure sets the legacy DNSExperimentFailure field.
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

// analysisDNSBogon computes the AnalysisDNSBogon flag. We set this flag if
// we dectect any bogon in the .Queries field of the TestKeys.
func (tk *TestKeys) analysisDNSBogon(logger model.Logger) {
	for _, query := range tk.Queries {
		for _, answer := range query.Answers {
			switch answer.AnswerType {
			case "A":
				if net.ParseIP(answer.IPv4) != nil && netxlite.IsBogon(answer.IPv4) {
					logger.Warnf("DNS: BOGON %s in #%d", answer.IPv4, query.TransactionID)
					tk.XDNSFlags |= AnalysisDNSBogon
					// continue processing so we print all the bogons we have
				}
			case "AAAA":
				if net.ParseIP(answer.IPv6) != nil && netxlite.IsBogon(answer.IPv6) {
					logger.Warnf("DNS: BOGON %s in #%d", answer.IPv6, query.TransactionID)
					tk.XDNSFlags |= AnalysisDNSBogon
					// continue processing so we print all the bogons we have
				}
			default:
				// nothing
			}
		}
	}
}

// analysisDNSUnexpectedFailure computes the AnalysisDNSUnexpectedFailure flags. We say
// a failure is unexpected when the TH could resolve a domain and the probe couldn't.
func (tk *TestKeys) analysisDNSUnexpectedFailure(logger model.Logger) {
	// make sure we have control before proceeding futher
	if tk.Control == nil || tk.ControlRequest == nil {
		return
	}

	// obtain thRequest and thResponse as shortcuts
	thRequest := tk.ControlRequest
	thResponse := tk.Control

	// obtain the domain that the TH has queried for
	URL, err := url.Parse(thRequest.HTTPRequest)
	if err != nil {
		return // this looks like a bug
	}
	domain := URL.Hostname()

	// we obviously don't care if the domain was an IP adddress
	if net.ParseIP(domain) != nil {
		return
	}

	// if the control didn't lookup any IP addresses our job here is done
	// because we can't say whether we have unexpected failures
	hasAddrs := len(thResponse.DNS.Addrs) > 0
	if !hasAddrs {
		return
	}

	// with TH-resolved addrs, any local query _for the same domain_ queried
	// by the probe that contains an error is suspicious
	for _, query := range tk.Queries {
		if domain != query.Hostname {
			continue // not the domain queried by the test helper
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
			// answers, so this seems a bug?
			continue
		}
		if query.QueryType == "AAAA" && *query.Failure == netxlite.FailureDNSNoAnswer {
			// maybe this heuristic could be further improved by checking
			// whether the TH did actually see any IPv6 address?
			continue
		}
		logger.Warnf("DNS: unexpected failure %s in #%d", *query.Failure, query.TransactionID)
		tk.XDNSFlags |= AnalysisDNSUnexpectedFailure
		// continue processing so we print all the unexpected failures
	}
}

// analysisDNSUnexpectedAddrs computes the AnalysisDNSUnexpectedAddrs flags. This
// algorithm builds upon the original DNSDiff algorithm by introducing an additional
// TLS based heuristic for determining whether an IP address was legit.
func (tk *TestKeys) analysisDNSUnexpectedAddrs(logger model.Logger) {
	// make sure we have control before proceeding futher
	if tk.Control == nil || tk.ControlRequest == nil {
		return
	}

	// obtain thRequest and thResponse as shortcuts
	thRequest := tk.ControlRequest
	thResponse := tk.Control

	// obtain the domain that the TH has queried for
	URL, err := url.Parse(thRequest.HTTPRequest)
	if err != nil {
		return // this looks like a bug
	}
	domain := URL.Hostname()

	// we obviously don't care if the domain was an IP adddress
	if net.ParseIP(domain) != nil {
		return
	}

	// if the control didn't resolve any IP address, then we basically
	// cannot run this algorithm at all
	thAddrs := thResponse.DNS.Addrs
	if len(thAddrs) <= 0 {
		return
	}

	// gather all the IP addresses queried by the probe
	// for the same domain for which the TH queried.
	var probeAddrs []string
	for _, query := range tk.Queries {
		if domain != query.Hostname {
			continue // not the domain the TH queried for
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
		tk.XDNSFlags |= AnalysisDNSUnexpectedAddrs
		return
	}

	// if there are no different addresses between the probe and the TH then
	// our job here is done and we can just stop searching
	differentAddrs := tk.analysisDNSDiffAddrs(probeAddrs, thAddrs)
	if len(differentAddrs) <= 0 {
		return
	}

	// now, let's exclude the differentAddrs for which we successfully
	// completed a TLS handshake: those should be good addrs
	withoutHandshake := tk.findAddrsWithoutTLSHandshake(domain, differentAddrs)
	if len(withoutHandshake) <= 0 {
		return
	}

	// as a last resort, accept the addresses without an handshake whose
	// ASN overlaps with ASNs resolved by the TH
	differentASNs := tk.analysisDNSDiffASN(withoutHandshake, thAddrs)
	if len(differentASNs) <= 0 {
		return
	}

	// otherwise, conclude we have unexpected probe addrs
	logger.Warnf(
		"DNSDiff: differentAddrs: %+v; withoutHandshake: %+v; differentASNs: %+v",
		differentAddrs, withoutHandshake, differentASNs,
	)
	tk.XDNSFlags |= AnalysisDNSUnexpectedAddrs
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
		if (where & inTH) == 0 {
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
		if (where & inTH) == 0 {
			asns = append(asns, asn)
		}
	}
	return
}

// findAddrsWithoutTLSHandshake computes the list of probe discovered [addresses]
// for which we couldn't successfully perform a TLS handshake for the given [domain].
func (tk *TestKeys) findAddrsWithoutTLSHandshake(domain string, addresses []string) (output []string) {
	const (
		resolved = 1 << iota
		handshakeOK
	)
	mapping := make(map[string]int)

	// fill the input map with the addresses we're interested to analyze
	for _, addr := range addresses {
		mapping[addr] = 0
	}

	// flag the subset of addresses resolved by the probe
	for _, query := range tk.Queries {
		for _, answer := range query.Answers {
			var addr string
			switch answer.AnswerType {
			case "A":
				addr = answer.IPv4
			case "AAAA":
				addr = answer.IPv6
			default:
				continue
			}
			if _, found := mapping[addr]; !found {
				continue // we're not interested into this addr
			}
			mapping[addr] |= resolved
		}
	}

	// flag the subset of addrs with successful handshake for the right SNI
	for _, thx := range tk.TLSHandshakes {
		addr, _, err := net.SplitHostPort(thx.Address)
		if err != nil {
			continue // looks like a bug
		}
		if thx.Failure != nil {
			continue // this handshake failed
		}
		if _, found := mapping[addr]; !found {
			continue // we're not interested into this addr
		}
		if thx.ServerName != domain {
			continue // the SNI is different, so...
		}
		mapping[addr] |= handshakeOK
	}

	// compute the list of addresses without the handshakeOK flag
	for addr, flags := range mapping {
		if flags == 0 {
			continue // this looks like a bug
		}
		if (flags & (resolved | handshakeOK)) == resolved {
			output = append(output, addr)
		}
	}
	return
}
