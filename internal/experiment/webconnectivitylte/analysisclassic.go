package webconnectivitylte

//
// The "classic" analysis engine.
//
// We try to emulate results produced by v0.4 of Web Connectivity and
// also attempt to provide a more fine-grained view of the results.
//

import (
	"net"

	"github.com/ooni/probe-cli/v3/internal/geoipx"
	"github.com/ooni/probe-cli/v3/internal/minipipeline"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/optional"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// analysisEngineClassic is an alternative analysis engine that aims to produce
// results that are backward compatible with Web Connectivity v0.4 while also
// procuding more fine-grained blocking flags.
func analysisEngineClassic(tk *TestKeys, logger model.Logger) {
	tk.analysisClassic(model.GeoIPASNLookupperFunc(geoipx.LookupASN), logger)
}

func (tk *TestKeys) analysisClassic(lookupper model.GeoIPASNLookupper, logger model.Logger) {
	// Since we run after all tasks have completed (or so we assume) we're
	// not going to use any form of locking here.

	// 1. produce web observations
	container := minipipeline.NewWebObservationsContainer()
	container.IngestDNSLookupEvents(lookupper, tk.Queries...)
	container.IngestTCPConnectEvents(lookupper, tk.TCPConnect...)
	container.IngestTLSHandshakeEvents(tk.TLSHandshakes...)
	container.IngestHTTPRoundTripEvents(tk.Requests...)

	// be defensive in case the control request or response are not defined
	if tk.ControlRequest != nil && tk.Control != nil {
		// Implementation note: the only error that can happen here is when the input
		// doesn't parse as a URL, which should have caused measurer.go to fail
		runtimex.Try0(container.IngestControlMessages(tk.ControlRequest, tk.Control))
	}

	// 2. compute extended analysis flags
	analysisExtMain(lookupper, tk, container)

	// 3. filter observations to only include results collected by the
	// system resolver, which approximates v0.4's results
	classic := minipipeline.ClassicFilter(container)

	// 3. produce a web observations analysis based on the web observations
	woa := minipipeline.AnalyzeWebObservationsWithLinearAnalysis(lookupper, classic)

	// 5. determine the DNS consistency
	tk.DNSConsistency = analysisClassicDNSConsistency(woa)

	// 6. set DNSExperimentFailure
	if !woa.DNSExperimentFailure.IsNone() && woa.DNSExperimentFailure.Unwrap() != "" {
		value := woa.DNSExperimentFailure.Unwrap()
		tk.DNSExperimentFailure = &value
	}

	// 7. compute the HTTPDiff values
	tk.setHTTPDiffValues(woa)

	// 8. compute blocking & accessible
	analysisClassicComputeBlockingAccessible(woa, tk)
}

func analysisClassicDNSConsistency(woa *minipipeline.WebAnalysis) optional.Value[string] {
	switch {
	case woa.DNSLookupUnexpectedFailure.Len() <= 0 && // no unexpected failures; and
		woa.DNSLookupSuccessWithInvalidAddressesClassic.Len() <= 0 && // no invalid addresses; and
		(woa.DNSLookupSuccessWithValidAddressClassic.Len() > 0 || // good addrs; or
			woa.DNSLookupExpectedFailure.Len() > 0): // expected failures
		return optional.Some("consistent")

	case woa.DNSLookupSuccessWithInvalidAddressesClassic.Len() > 0 || // unexpected addrs; or
		woa.DNSLookupUnexpectedFailure.Len() > 0 || // unexpected failures; or
		(woa.DNSLookupSuccess.Len() > 0 && // successful lookups; and
			!woa.ControlExpectations.IsNone() && // we have control info; and
			woa.ControlExpectations.Unwrap().DNSAddresses.Len() <= 0): // control resolved nothing
		return optional.Some("inconsistent")

	default:
		return optional.None[string]() // none of the above
	}
}

func (tk *TestKeys) setHTTPDiffValues(woa *minipipeline.WebAnalysis) {
	hds := newAnalysisHTTPDiffStatus(woa)
	tk.BodyProportion = hds.BodyProportion.UnwrapOr(0)
	tk.BodyLengthMatch = hds.BodyLengthMatch
	tk.HeadersMatch = hds.HeadersMatch
	tk.StatusCodeMatch = hds.StatusCodeMatch
	tk.TitleMatch = hds.TitleMatch
}

type analysisClassicTestKeysProxy interface {
	// httpDiff returns true if there's an http-diff.
	httpDiff() bool

	// setBlockingString sets blocking to a string.
	setBlockingString(value string)

	// setBlockingNil sets blocking to nil.
	setBlockingNil()

	// setBlockingFalse sets Blocking to false.
	setBlockingFalse()

	// setHTTPExperimentFailure sets the HTTPExperimentFailure field.
	setHTTPExperimentFailure(value optional.Value[string])

	// setWebsiteDown sets the test keys for a down website.
	setWebsiteDown()
}

var _ analysisClassicTestKeysProxy = &TestKeys{}

// httpDiff implements analysisClassicTestKeysProxy.
func (tk *TestKeys) httpDiff() bool {
	return analysisHTTPDiffAlgorithm(tk)
}

// bodyLengthMatch implements analysisHTTPDiffValuesProvider.
func (tk *TestKeys) bodyLengthMatch() optional.Value[bool] {
	return tk.BodyLengthMatch
}

// headersMatch implements analysisHTTPDiffValuesProvider.
func (tk *TestKeys) headersMatch() optional.Value[bool] {
	return tk.HeadersMatch
}

// statusCodeMatch implements analysisHTTPDiffValuesProvider.
func (tk *TestKeys) statusCodeMatch() optional.Value[bool] {
	return tk.StatusCodeMatch
}

// titleMatch implements analysisHTTPDiffValuesProvider.
func (tk *TestKeys) titleMatch() optional.Value[bool] {
	return tk.TitleMatch
}

// setBlockingFalse implements analysisClassicTestKeysProxy.
func (tk *TestKeys) setBlockingFalse() {
	tk.Blocking = false
	tk.Accessible = true
}

// setBlockingNil implements analysisClassicTestKeysProxy.
func (tk *TestKeys) setBlockingNil() {
	if !tk.DNSConsistency.IsNone() && tk.DNSConsistency.Unwrap() == "inconsistent" {
		tk.Blocking = "dns"
		tk.Accessible = false
	} else {
		tk.Blocking = nil
		tk.Accessible = nil
	}
}

// setBlockingString implements analysisClassicTestKeysProxy.
func (tk *TestKeys) setBlockingString(value string) {
	if !tk.DNSConsistency.IsNone() && tk.DNSConsistency.Unwrap() == "inconsistent" {
		tk.Blocking = "dns"
	} else {
		tk.Blocking = value
	}
	tk.Accessible = false
}

// setHTTPExperimentFailure implements analysisClassicTestKeysProxy.
func (tk *TestKeys) setHTTPExperimentFailure(value optional.Value[string]) {
	tk.HTTPExperimentFailure = value
}

// setWebsiteDown implements analysisClassicTestKeysProxy.
func (tk *TestKeys) setWebsiteDown() {
	if !tk.DNSConsistency.IsNone() && tk.DNSConsistency.Unwrap() == "inconsistent" {
		tk.Blocking = "dns"
		tk.Accessible = false
	} else {
		tk.Blocking = false
		tk.Accessible = false
	}
}

func analysisClassicComputeBlockingAccessible(woa *minipipeline.WebAnalysis, tk analysisClassicTestKeysProxy) {
	// minipipeline.NewLinearWebAnalysis produces a woa.Linear sorted
	//
	// 1. by descending TagDepth;
	//
	// 2. with TagDepth being equal, by descending [WebObservationType];
	//
	// 3. with [WebObservationType] being equal, by ascending failure string;
	//
	// This means that you divide the list in groups like this:
	//
	//	+------------+------------+------------+------------+
	//	| TagDepth=3 | TagDepth=2 | TagDepth=1 | TagDepth=0 |
	//	+------------+------------+------------+------------+
	//
	// Where TagDepth=3 is the last redirect and TagDepth=0 is the initial request.
	//
	// Each group is further divided as follows:
	//
	//	+------+-----+-----+-----+
	//	| HTTP | TLS | TCP | DNS |
	//	+------+-----+-----+-----+
	//
	// Where each group may be empty. The first non-empty group is about the
	// operation that failed for the current TagDepth.
	//
	// Within each group, successes sort before failures because the empty
	// string has priority over non-empty strings.
	//
	// So, when walking the list from index 0 to index N, you encounter the
	// latest redirects first, you observe the more complex operations first,
	// and you see errors before failures.
	for _, entry := range woa.Linear {

		// 1. As a special case, handle a "final" response first. We define "final" a
		// successful response whose status code is like 2xx, 4xx, or 5xx.
		if !entry.HTTPResponseIsFinal.IsNone() && entry.HTTPResponseIsFinal.Unwrap() {

			// 1.1. Handle the case of succesful response over TLS.
			if !entry.TLSHandshakeFailure.IsNone() && entry.TLSHandshakeFailure.Unwrap() == "" {
				tk.setBlockingFalse()
				return
			}

			// 1.2. Handle the case of missing HTTP control.
			if entry.ControlHTTPFailure.IsNone() {
				tk.setBlockingNil()
				return
			}

			// 1.3. Figure out whether the measurement and the control are close enough.
			if !tk.httpDiff() {
				tk.setBlockingFalse()
				return
			}

			// 1.4. There's something different in the two responses.
			tk.setBlockingString("http-diff")
			return
		}

		// 2. Let's now focus on failed HTTP round trips.
		if entry.Type == minipipeline.WebObservationTypeHTTPRoundTrip &&
			!entry.Failure.IsNone() && entry.Failure.Unwrap() != "" {

			// 2.1. Handle the case of a missing HTTP control. Maybe
			// the control server is unreachable or blocked.
			if entry.ControlHTTPFailure.IsNone() {
				tk.setBlockingNil()
				tk.setHTTPExperimentFailure(entry.Failure)
				return
			}

			// 2.2. Handle the case where both the probe and the control failed.
			if entry.ControlHTTPFailure.Unwrap() != "" {
				tk.setWebsiteDown()
				tk.setHTTPExperimentFailure(entry.Failure)
				return
			}

			// 2.3. Handle the case where just the probe failed.
			tk.setBlockingString("http-failure")
			tk.setHTTPExperimentFailure(entry.Failure)
			return
		}

		// 3. Handle the case of TLS failure.
		if entry.Type == minipipeline.WebObservationTypeTLSHandshake &&
			!entry.Failure.IsNone() && entry.Failure.Unwrap() != "" {

			// 3.1. Handle the case of missing TLS control information. The control
			// only provides information for the first request. Once we start following
			// redirects we do not have TLS/TCP/DNS control.
			if entry.ControlTLSHandshakeFailure.IsNone() {

				// 3.1.1 Handle the case of missing an expectation about what
				// accessing the website should lead to, which is set forth by
				// the control accessing the website and telling us.
				if entry.ControlHTTPFailure.IsNone() {
					tk.setBlockingNil()
					tk.setHTTPExperimentFailure(entry.Failure)
					return
				}

				// 3.1.2. Otherwise, if the control worked, that's blocking.
				tk.setBlockingString("http-failure")
				tk.setHTTPExperimentFailure(entry.Failure)
				return
			}

			// 3.2. Handle the case where both probe and control failed.
			if entry.ControlTLSHandshakeFailure.Unwrap() != "" {
				tk.setWebsiteDown()
				tk.setHTTPExperimentFailure(entry.Failure)
				return
			}

			// 3.3. Handle the case where just the probe failed.
			tk.setBlockingString("http-failure")
			tk.setHTTPExperimentFailure(entry.Failure)
			return
		}

		// 4. Handle the case of TCP failure.
		if entry.Type == minipipeline.WebObservationTypeTCPConnect &&
			!entry.Failure.IsNone() && entry.Failure.Unwrap() != "" {

			// 4.1. Handle the case of missing TCP control info.
			if entry.ControlTCPConnectFailure.IsNone() {

				// 4.1.1 Handle the case of missing an expectation about what
				// accessing the website should lead to.
				if entry.ControlHTTPFailure.IsNone() {
					tk.setBlockingNil()
					tk.setHTTPExperimentFailure(entry.Failure)
					return
				}

				// 4.1.2. Otherwise, if the control worked, that's blocking.
				tk.setBlockingString("http-failure")
				tk.setHTTPExperimentFailure(entry.Failure)
				return
			}

			// 4.2. Handle the case where both probe and control failed.
			if entry.ControlTCPConnectFailure.Unwrap() != "" {
				tk.setWebsiteDown()
				tk.setHTTPExperimentFailure(entry.Failure)
				return
			}

			// 4.3. Handle the case where just the probe failed.
			tk.setBlockingString("tcp_ip")
			tk.setHTTPExperimentFailure(entry.Failure)
			return
		}

		// 5. Handle the case of DNS failure
		if entry.Type == minipipeline.WebObservationTypeDNSLookup &&
			!entry.Failure.IsNone() && entry.Failure.Unwrap() != "" {

			// 5.1. Handle the case of missing DNS control info.
			if entry.ControlDNSLookupFailure.IsNone() {

				// 5.1.1 Handle the case of missing an expectation about what
				// accessing the website should lead to.
				if entry.ControlHTTPFailure.IsNone() {
					tk.setBlockingFalse()
					tk.setHTTPExperimentFailure(entry.Failure)
					return
				}

				// 5.1.2. Otherwise, if the control worked, that's blocking.
				tk.setBlockingString("dns")
				tk.setHTTPExperimentFailure(entry.Failure)
				return
			}

			// 5.2. Handle the case where both probe and control failed.
			if entry.ControlDNSLookupFailure.Unwrap() != "" {
				tk.setWebsiteDown()
				tk.setHTTPExperimentFailure(entry.Failure)
				return
			}

			// 5.3. special case for when the probe failed with dns_no_data and the
			// test helper returned no addresses, which is its way to tell us the same
			// error occurred but we need to interpret it as dns_no_data.
			if !entry.ControlDNSResolvedAddrs.IsNone() &&
				entry.ControlDNSResolvedAddrs.Unwrap().Len() <= 0 {
				tk.setWebsiteDown()
				return
			}

			// 5.4. Handle the case where just the probe failed.
			tk.setBlockingString("dns")
			tk.setHTTPExperimentFailure(entry.Failure)
			return
		}

		// 6. handle the case of DNS success with the probe only seeing loopback
		// addrs while the TH sees real addresses, which is a case where in the
		// classic analysis (which is what we're doing) the probe does not attempt
		// to connect to loopback addresses because it doesn't make sense.
		if entry.Type == minipipeline.WebObservationTypeDNSLookup &&
			!entry.Failure.IsNone() && entry.Failure.Unwrap() == "" &&
			!entry.ControlDNSLookupFailure.IsNone() &&
			entry.ControlDNSLookupFailure.Unwrap() == "" &&
			!entry.DNSResolvedAddrs.IsNone() && !entry.ControlDNSResolvedAddrs.IsNone() &&
			analysisContainsOnlyLoopbackAddrs(entry.DNSResolvedAddrs.Unwrap()) &&
			!analysisContainsOnlyLoopbackAddrs(entry.ControlDNSResolvedAddrs.Unwrap()) {
			tk.setBlockingString("dns")
			return
		}

		// 7. handle the case of DNS success with loopback addrs, which is the case
		// where neither the probe nor the TH attempt to measure endpoints.
		if entry.Type == minipipeline.WebObservationTypeDNSLookup &&
			!entry.Failure.IsNone() && entry.Failure.Unwrap() == "" &&
			!entry.ControlDNSLookupFailure.IsNone() &&
			entry.ControlDNSLookupFailure.Unwrap() == "" &&
			!entry.DNSResolvedAddrs.IsNone() && !entry.ControlDNSResolvedAddrs.IsNone() &&
			analysisContainsOnlyLoopbackAddrs(entry.DNSResolvedAddrs.Unwrap()) &&
			analysisContainsOnlyLoopbackAddrs(entry.ControlDNSResolvedAddrs.Unwrap()) {
			tk.setWebsiteDown()
			return
		}
	}
}

// analysisContainsOnlyLoopbackAddrs returns true iff the given set contains one or
// more IP addresses and all these adresses are loopback addresses.
func analysisContainsOnlyLoopbackAddrs(addrs minipipeline.Set[string]) bool {
	var count int
	for _, addr := range addrs.Keys() {
		if net.ParseIP(addr) == nil {
			continue
		}
		if !netxlite.IsLoopback(addr) {
			return false
		}
		count++
	}
	return count > 0
}
