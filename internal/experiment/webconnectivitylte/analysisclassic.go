package webconnectivitylte

import (
	"fmt"

	"github.com/ooni/probe-cli/v3/internal/minipipeline"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/optional"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func (tk *TestKeys) analysisClassic(logger model.Logger) {
	// Since we run after all tasks have completed (or so we assume) we're
	// not going to use any form of locking here.

	// 1. produce observations using the minipipeline
	container := minipipeline.NewWebObservationsContainer()
	container.IngestDNSLookupEvents(tk.Queries...)
	container.IngestTCPConnectEvents(tk.TCPConnect...)
	container.IngestTLSHandshakeEvents(tk.TLSHandshakes...)
	container.IngestHTTPRoundTripEvents(tk.Requests...)

	// be defensive in case the control request or control are not defined
	if tk.ControlRequest != nil && tk.Control != nil {
		// Implementation note: the only error that can happen here is when the input
		// doesn't parse as a URL, which should have triggered previous errors
		runtimex.Try0(container.IngestControlMessages(tk.ControlRequest, tk.Control))
	}

	// 2. filter observations to only include results collected by the
	// system resolver, which approximates v0.4's results
	classic := minipipeline.ClassicFilter(container)

	// dump the observations
	fmt.Printf("%s\n", must.MarshalJSON(classic))

	// 3. produce the woa based on the observations
	woa := minipipeline.AnalyzeWebObservations(classic)

	// dump the analysis
	fmt.Printf("%s\n", must.MarshalJSON(woa))

	// 4. determine the DNS consistency
	switch {
	case woa.DNSLookupUnexpectedFailure.Len() <= 0 && // no unexpected failures; and
		woa.DNSLookupSuccessWithInvalidAddressesClassic.Len() <= 0 && // no invalid addresses; and
		(woa.DNSLookupSuccessWithValidAddressClassic.Len() > 0 || // good addrs; or
			woa.DNSLookupExpectedFailure.Len() > 0): // expected failures
		tk.DNSConsistency = optional.Some("consistent")

	case woa.DNSLookupSuccessWithInvalidAddressesClassic.Len() > 0 || // unexpected addrs; or
		woa.DNSLookupUnexpectedFailure.Len() > 0: // unexpected failures
		tk.DNSConsistency = optional.Some("inconsistent")

	default:
		tk.DNSConsistency = optional.None[string]()
	}

	// we must set blocking to "dns" when there's a DNS inconsistency
	setBlocking := func(value string) string {
		switch {
		case !tk.DNSConsistency.IsNone() && tk.DNSConsistency.Unwrap() == "inconsistent":
			return "dns"
		default:
			return value
		}
	}

	setBlockingNil := func() {
		switch {
		case !tk.DNSConsistency.IsNone() && tk.DNSConsistency.Unwrap() == "inconsistent":
			tk.Blocking = "dns"
			tk.Accessible = false
		default:
			tk.Blocking = nil
			tk.Accessible = nil
		}
	}

	// 5. set HTTPDiff values
	if !woa.HTTPFinalResponseDiffBodyProportionFactor.IsNone() {
		tk.BodyLengthMatch = optional.Some(woa.HTTPFinalResponseDiffBodyProportionFactor.Unwrap() > 0.7)
	}
	if !woa.HTTPFinalResponseDiffUncommonHeadersIntersection.IsNone() {
		tk.HeadersMatch = optional.Some(len(woa.HTTPFinalResponseDiffUncommonHeadersIntersection.Unwrap()) > 0)
	}
	tk.StatusCodeMatch = woa.HTTPFinalResponseDiffStatusCodeMatch
	if !woa.HTTPFinalResponseDiffTitleDifferentLongWords.IsNone() {
		tk.TitleMatch = optional.Some(len(woa.HTTPFinalResponseDiffTitleDifferentLongWords.Unwrap()) <= 0)
	}

	// 6. determine blocking & accessible

	for _, entry := range woa.Linear {

		// handle the case where there's a final response
		//
		// as a reminder, a final response is a successful response with 2xx, 4xx or 5xx status
		if !entry.HTTPResponseIsFinal.IsNone() && entry.HTTPResponseIsFinal.Unwrap() {

			// if we were using TLS, we're good
			if !entry.TLSHandshakeFailure.IsNone() && entry.TLSHandshakeFailure.Unwrap() == "" {
				tk.Blocking = false
				tk.Accessible = true
				return
			}

			// handle the case where there's no control
			if entry.ControlHTTPFailure.IsNone() {
				tk.Blocking = nil
				tk.Accessible = nil
				return
			}

			// try to determine whether the page is good
			if !tk.StatusCodeMatch.IsNone() && tk.StatusCodeMatch.Unwrap() {
				if !tk.BodyLengthMatch.IsNone() && tk.BodyLengthMatch.Unwrap() {
					tk.Blocking = false
					tk.Accessible = true
					return
				}
				if !tk.HeadersMatch.IsNone() && tk.HeadersMatch.Unwrap() {
					tk.Blocking = false
					tk.Accessible = true
					return
				}
				if !tk.TitleMatch.IsNone() && tk.TitleMatch.Unwrap() {
					tk.Blocking = false
					tk.Accessible = true
					return
				}
				// fallthrough
			}

			// otherwise, declare that there's an http-diff
			tk.Blocking = setBlocking("http-diff")
			tk.Accessible = false
			return
		}

		// handle the case of HTTP failure
		if entry.Type == minipipeline.WebObservationTypeHTTPRoundTrip &&
			!entry.Failure.IsNone() && entry.Failure.Unwrap() != "" {

			// handle the case where there's no control
			if entry.ControlHTTPFailure.IsNone() {
				tk.Blocking = nil
				tk.Accessible = nil
				return
			}

			// handle the case of expected failure
			if entry.ControlHTTPFailure.Unwrap() != "" {
				// TODO(bassosimone): this is wrong but Web Connectivity v0.4 does not
				// correctly distinguish down websites, so we need to do the same for
				// comparability purposes. The correct result is with .Accessible == false.
				tk.Blocking = false
				tk.Accessible = true
				return
			}

			// handle an unexpected failure
			tk.Blocking = setBlocking("http-failure")
			tk.Accessible = false
			return
		}

		// handle the case of TLS failure
		if entry.Type == minipipeline.WebObservationTypeTLSHandshake &&
			!entry.Failure.IsNone() && entry.Failure.Unwrap() != "" {

			// handle the case where there is no TLS control
			if entry.ControlTLSHandshakeFailure.IsNone() {

				// handle the case where we have no final expectation from the control
				if entry.ControlHTTPFailure.IsNone() {
					tk.Blocking = nil
					tk.Accessible = nil
					return
				}

				// otherwise, we're probably in a redirect w/o control
				tk.Blocking = setBlocking("http-failure")
				tk.Accessible = false
				return
			}

			// handle the case of expected failure
			if entry.ControlTLSHandshakeFailure.Unwrap() != "" {
				// TODO(bassosimone): this is wrong but Web Connectivity v0.4 does not
				// correctly distinguish down websites, so we need to do the same for
				// comparability purposes. The correct result here is false, false.
				//
				// XXX also this algorithm here is disgusting
				setBlockingNil()
				return
			}

			// handle an unexpected failure
			tk.Blocking = setBlocking("http-failure")
			tk.Accessible = false
			return
		}

		// handle the case of TCP failure
		if entry.Type == minipipeline.WebObservationTypeTCPConnect &&
			!entry.Failure.IsNone() && entry.Failure.Unwrap() != "" {

			// handle the case where there's no TCP control
			if entry.ControlTCPConnectFailure.IsNone() {

				// handle the case where we have no final expectation from the control
				if entry.ControlHTTPFailure.IsNone() {
					tk.Blocking = nil
					tk.Accessible = nil
					return
				}

				// otherwise, we're probably in a redirect w/o control
				tk.Blocking = setBlocking("http-failure")
				tk.Accessible = false
				return
			}

			// handle the case of expected failure
			if entry.ControlTCPConnectFailure.Unwrap() != "" {
				// TODO(bassosimone): this is wrong but Web Connectivity v0.4 does not
				// correctly distinguish down websites, so we need to do the same for
				// comparability purposes. The correct result is with .Accessible == false.
				tk.Blocking = false
				tk.Accessible = true
				return
			}

			// handle an unexpected failure
			tk.Blocking = setBlocking("tcp_ip")
			tk.Accessible = false
			return
		}

		// handle the case of DNS failure
		if entry.Type == minipipeline.WebObservationTypeDNSLookup &&
			!entry.Failure.IsNone() && entry.Failure.Unwrap() != "" {

			// handle the case where there's no DNS control
			if entry.ControlDNSLookupFailure.IsNone() {

				// BUG: we need to copy control information also for DNS
				// lookups otherwise we're not able to xcompare.

				// handle the case where we have no final expectation from the control
				if entry.ControlHTTPFailure.IsNone() {
					tk.Blocking = nil
					tk.Accessible = nil
					return
				}

				// otherwise, we're probably in a redirect w/o control
				tk.Blocking = setBlocking("dns")
				tk.Accessible = false
				return
			}

			// handle the case of expected failure
			if entry.ControlDNSLookupFailure.Unwrap() != "" {
				// TODO(bassosimone): this is wrong but Web Connectivity v0.4 does not
				// correctly distinguish down websites, so we need to do the same for
				// comparability purposes. The correct result is with .Accessible == false.
				tk.Blocking = false
				tk.Accessible = true
				return
			}

			// handle an unexpected failure
			tk.Blocking = setBlocking("dns")
			tk.Accessible = false
			return
		}
	}

	// if we arrive here, it means we could not make sense of what
	// happened, hence let us ask for help to our users

	logger.Warnf("BUG! We were not able to classify this measurement!")
	logger.Warnf("The following is the list of observations we were processing:")
	logger.Warnf("\n%s", must.MarshalAndIndentJSON(woa, "", " "))
	logger.Warnf("Please, report this bug at https://github.com/ooni/probe/issues/new")
	logger.Warnf("including the above JSON and possibly additional context")
}
