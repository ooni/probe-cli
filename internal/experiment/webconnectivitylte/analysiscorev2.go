package webconnectivitylte

import (
	"fmt"

	"github.com/ooni/probe-cli/v3/internal/minipipeline"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// analysisToplevelV2 is an alternative version of the analysis code that
// uses the [minipipeline] package for processing.
func (tk *TestKeys) analysisToplevelV2(logger model.Logger) {
	// Since we run after all tasks have completed (or so we assume) we're
	// not going to use any form of locking here.

	container := minipipeline.NewWebObservationsContainer()
	container.CreateDNSLookupFailures(tk.Queries...)
	container.CreateKnownIPAddresses(tk.Queries...)
	container.CreateKnownTCPEndpoints(tk.TCPConnect...)
	container.NoteTLSHandshakeResults(tk.TLSHandshakes...)
	container.NoteHTTPRoundTripResults(tk.Requests...)

	// be defensive in case the control request or control are not defined
	if tk.ControlRequest != nil && tk.Control != nil {
		// Implementation note: the only error that can happen here is when the input
		// doesn't parse as a URL, which should have triggered previous errors
		runtimex.Try0(container.NoteControlResults(tk.ControlRequest, tk.Control))
	}

	// dump the pipeline results for debugging purposes
	fmt.Printf("%s\n", must.MarshalJSON(container))

	// run top-level protocol-specific analysis algorithms
	tk.analysisDNSToplevelV2(container, logger)
	tk.analysisTCPIPToplevelV2(container, logger)
	tk.analysisTLSToplevelV2(container, logger)
	tk.analysisHTTPToplevelV2(container, logger)
}

// analysisDNSToplevelV2 is the toplevel analysis function for DNS results.
//
// Note: this function DOES NOT consider failed DNS-over-HTTPS (DoH) submeasurements
// and ONLY considers the IP addrs they have resolved. Failing to contact a DoH service
// provides info about such a DoH service rather than on the measured URL. See the
// https://github.com/ooni/probe/issues/2274 issue for more info.
//
// The goals of this function are the following:
//
// 1. Set the legacy .DNSExperimentFailure field to the failure value of the
// first DNS query that failed among the ones using getaddrinfo. This field is
// legacy because now we perform several DNS lookups.
//
// 2. Compute the XDNSFlags value.
//
// From the XDNSFlags value, we determine, in turn DNSConsistency and
// XBlockingFlags according to the following decision table:
//
//	+-----------+----------------+---------------------+
//	| XDNSFlags | DNSConsistency | XBlockingFlags      |
//	+-----------+----------------+---------------------+
//	| 0         | "consistent"   | no change           |
//	+-----------+----------------+---------------------+
//	| nonzero   | "inconsistent" | set FlagDNSBlocking |
//	+-----------+----------------+---------------------+
//
// We explain how XDNSFlags is determined in the documentation of
// the functions that this function calls to do its job.
func (tk *TestKeys) analysisDNSToplevelV2(container *minipipeline.WebObservationsContainer, logger model.Logger) {

	// run the analysis algorithms
	tk.analysisDNSExperimentFailureV2(container)
	tk.analysisDNSBogonV2(container, logger)
	//tk.analysisDNSDuplicateResponsesV2() // TODO(bassosimone): implement
	tk.analysisDNSUnexpectedFailureV2(container, logger)
	tk.analysisDNSUnexpectedAddrsV2(container, logger)

	// compute DNS consistency
	if tk.DNSFlags != 0 {
		logger.Warn("DNSConsistency: inconsistent")
		tk.DNSConsistency = "inconsistent"
		tk.BlockingFlags |= analysisFlagDNSBlocking
	} else {
		logger.Info("DNSConsistency: consistent")
		tk.DNSConsistency = "consistent"
	}

}

func (tk *TestKeys) analysisDNSExperimentFailureV2(container *minipipeline.WebObservationsContainer) {
	for _, obs := range container.DNSLookupFailures {
		// skip if we don't know the engine
		if obs.DNSEngine.IsNone() {
			continue
		}

		// make sure we only include the system resolver
		switch obs.DNSEngine.Unwrap() {
		case "getaddrinfo", "golang_net_resolver":

			// skip cases where the failure is not set, which is a bug
			if obs.DNSLookupFailure.IsNone() {
				continue
			}
			failure := obs.DNSLookupFailure.Unwrap()

			// skip cases where the query type is not set, which is a bug
			if obs.DNSQueryType.IsNone() {
				continue
			}
			queryType := obs.DNSQueryType.Unwrap()

			// skip cases where there's no DNS record for AAAA
			if queryType == "AAAA" && failure == netxlite.FailureDNSNoAnswer {
				continue
			}

			// set the failure proper
			tk.DNSExperimentFailure = &failure

		default:
			// nothing
		}
	}
}

func (tk *TestKeys) analysisDNSBogonV2(container *minipipeline.WebObservationsContainer, logger model.Logger) {
	// Implementation note: any bogon IP address resolved by a DoH service
	// is STILL suspicious since it should not happen. TODO(bassosimone): an
	// even better algorithm could possibly check whether also the TH has
	// observed bogon IP addrs and avoid flagging in such a case.
	//
	// See https://github.com/ooni/probe/issues/2274 for more information.

	for _, obs := range container.KnownTCPEndpoints {
		// skip cases where there's no bogon
		if obs.IPAddressBogon.IsNone() {
			continue
		}
		if !obs.IPAddressBogon.Unwrap() {
			continue
		}

		// skip cases where the IP address is not defined (likely a bug)
		if obs.IPAddress.IsNone() {
			continue
		}
		addr := obs.IPAddress.Unwrap()

		// skip cases where the domain is not known (likely a bug)
		if obs.DNSDomain.IsNone() {
			continue
		}
		domain := obs.DNSDomain.Unwrap()

		// log and make sure we set the correct flag
		logger.Warnf("DNS: got BOGON answer %s for domain %s (see %v)", addr, domain, obs.DNSTransactionIDs)
		tk.DNSFlags |= AnalysisDNSBogon

		// continue processing so we print all the bogons we have
	}
}

func (tk *TestKeys) analysisDNSUnexpectedFailureV2(container *minipipeline.WebObservationsContainer, logger model.Logger) {
	for _, obs := range container.DNSLookupFailures {
		// skip cases with no failures
		if obs.DNSLookupFailure.IsNone() {
			continue
		}
		failure := obs.DNSLookupFailure.Unwrap()
		if failure == "" {
			continue
		}

		// skip cases where also the control failed
		if obs.ControlDNSLookupFailure.IsNone() {
			continue
		}
		if obs.ControlDNSLookupFailure.Unwrap() != "" {
			continue
		}

		// A DoH failure is not information about the URL we're measuring
		// but about the DoH service being blocked.
		//
		// See https://github.com/ooni/probe/issues/2274
		if obs.DNSEngine.IsNone() {
			continue
		}
		engine := obs.DNSEngine.Unwrap()
		if engine == "doh" {
			continue
		}

		// skip cases where the query type is not set, which is a bug
		if obs.DNSQueryType.IsNone() {
			continue
		}
		queryType := obs.DNSQueryType.Unwrap()

		// skip cases where there's no DNS record for AAAA
		if queryType == "AAAA" && failure == netxlite.FailureDNSNoAnswer {
			continue
		}

		// log and make sure we set the correct flag
		logger.Warnf("DNS: unexpected failure %s in %v", failure, obs.DNSTransactionIDs)
		tk.DNSFlags |= AnalysisDNSUnexpectedFailure

		// continue processing so we print all the unexpected failures
	}
}

func (tk *TestKeys) analysisDNSUnexpectedAddrsV2(container *minipipeline.WebObservationsContainer, logger model.Logger) {
	// Implementation note: in the case in which DoH returned answers, here
	// it still feels okay to consider them. We should avoid flagging DoH
	// failures as measurement failures but if DoH returns us some unexpected
	// even-non-bogon addr, it seems worth flagging for now.
	//
	// See https://github.com/ooni/probe/issues/2274

	for _, obs := range container.KnownTCPEndpoints {
		// skip the cases with no address (which would be a bug)
		if obs.IPAddress.IsNone() {
			continue
		}
		addr := obs.IPAddress.Unwrap()

		// if the address was also resolved by the control, we're good
		if !obs.MatchWithControlIPAddress.IsNone() {
			if obs.MatchWithControlIPAddress.Unwrap() {
				continue
			}
			logger.Infof("DNS: address %s: not resolved by TH", addr)
		}

		// if we have a succesful TLS handshake for this addr, we're good
		//
		// note: this check is before the ASN check to avoid emitting a
		// warning indicating an ASN mismatch when we do not have an ASN
		if !obs.TLSHandshakeFailure.IsNone() {
			if obs.TLSHandshakeFailure.Unwrap() == "" {
				continue
			}
			logger.Infof("DNS: address %s: cannot confirm using TLS handshake", addr)
		}

		// we must have a valid ASN at this point (or bug!)
		if obs.IPAddressASN.IsNone() {
			continue
		}
		asn := obs.IPAddressASN.Unwrap()

		// if the ASN was also observed by the control, we're good
		if obs.MatchWithControlIPAddressASN.IsNone() {
			continue
		}
		if obs.MatchWithControlIPAddressASN.Unwrap() {
			continue
		}

		// log and make sure we set the correct flag
		logger.Warnf(
			"DNS: address %s has unexpected AS%d and we cannot use TLS to confirm it",
			addr, asn,
		)
		tk.DNSFlags |= AnalysisDNSUnexpectedAddrs

		// continue processing so we print all the unexpected failures
	}
}

// analysisTCPIPToplevelV2 is the toplevel analysis function for TCP/IP results.
func (tk *TestKeys) analysisTCPIPToplevelV2(container *minipipeline.WebObservationsContainer, logger model.Logger) {
	for _, obs := range container.KnownTCPEndpoints {
		// skip cases with no failures
		if obs.TCPConnectFailure.IsNone() {
			continue
		}
		failure := obs.TCPConnectFailure.Unwrap()
		if failure == "" {
			continue
		}

		// skip cases where also the control failed
		if obs.ControlTCPConnectFailure.IsNone() {
			continue
		}
		if obs.ControlTCPConnectFailure.Unwrap() != "" {
			continue
		}

		// TODO(bassosimone): how do we set the .Blocked flag?
		// maybe we can use the transactionID to go back to the right
		// data structure and set the value or we can deprecate it
		// and just ignore this corner case?

		// log and make sure we set the correct flag
		logger.Warnf(
			"TCP/IP: unexpected failure %s for %s (see %d)",
			failure,
			obs.EndpointAddress,
			obs.EndpointTransactionID,
		)
		tk.BlockingFlags |= analysisFlagTCPIPBlocking

		// continue processing so we print all the unexpected failures
	}
}

// analysisTLSToplevelV2 is the toplevel analysis function for TLS results.
func (tk *TestKeys) analysisTLSToplevelV2(container *minipipeline.WebObservationsContainer, logger model.Logger) {
	for _, obs := range container.KnownTCPEndpoints {
		// skip cases with no failures
		if obs.TLSHandshakeFailure.IsNone() {
			continue
		}
		failure := obs.TLSHandshakeFailure.Unwrap()
		if failure == "" {
			continue
		}

		// skip cases where also the control failed
		if obs.ControlTLSHandshakeFailure.IsNone() {
			continue
		}
		if obs.ControlTLSHandshakeFailure.Unwrap() != "" {
			continue
		}

		// log and make sure we set the correct flag
		logger.Warnf(
			"TLS: unexpected failure %s for %s (see %d)",
			failure,
			obs.EndpointAddress,
			obs.EndpointTransactionID,
		)
		tk.BlockingFlags |= analysisFlagTLSBlocking

		// continue processing so we print all the unexpected failures
	}
}

// analysisHTTPToplevelV2 is the toplevel analysis function for HTTP results.
func (tk *TestKeys) analysisHTTPToplevelV2(container *minipipeline.WebObservationsContainer, logger model.Logger) {
	tk.analysisHTTPUnexpectedFailureV2(container, logger)
	tk.analysisHTTPDiffV2(container, logger)
}

func (tk *TestKeys) analysisHTTPUnexpectedFailureV2(container *minipipeline.WebObservationsContainer, logger model.Logger) {
	for _, obs := range container.KnownTCPEndpoints {
		// skip cases with no failures
		if obs.HTTPFailure.IsNone() {
			continue
		}
		failure := obs.HTTPFailure.Unwrap()
		if failure == "" {
			continue
		}

		// skip cases where also the control failed
		if obs.ControlHTTPFailure.IsNone() {
			continue
		}
		if obs.ControlHTTPFailure.UnwrapOr("") != "" {
			continue
		}

		// log and make sure we set the correct flag
		logger.Warnf(
			"TLS: unexpected failure %s for %s (see %d)",
			failure,
			obs.EndpointAddress,
			obs.EndpointTransactionID,
		)
		tk.BlockingFlags |= analysisFlagTLSBlocking

		// continue processing so we print all the unexpected failures
	}
}

func (tk *TestKeys) analysisHTTPDiffV2(container *minipipeline.WebObservationsContainer, logger model.Logger) {
	for _, obs := range container.KnownTCPEndpoints {
		// skip cases with failures
		if obs.HTTPFailure.IsNone() {
			continue
		}
		failure := obs.HTTPFailure.Unwrap()
		if failure != "" {
			continue
		}

		// skip cases where the control failed
		if obs.ControlHTTPFailure.IsNone() {
			continue
		}
		if obs.ControlHTTPFailure.UnwrapOr("") != "" {
			continue
		}

	}
}
