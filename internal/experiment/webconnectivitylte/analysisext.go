package webconnectivitylte

//
// The extended ("ext") analysis sub-engine (used by "classic").
//
// We analyze all the produced observations without limiting ourselves to
// analyzing observations rooted into getaddrinfo lookups.
//

import (
	"fmt"
	"io"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/minipipeline"
)

// analysisExtMain computes the extended analysis.
//
// This function MUTATES the [*TestKeys].
func analysisExtMain(tk *TestKeys, container *minipipeline.WebObservationsContainer) {
	// compute the web analysis
	analysis := minipipeline.AnalyzeWebObservationsWithoutLinearAnalysis(container)

	// prepare for emitting informational messages
	var info strings.Builder

	// DNS & address analysis matching with control info (i.e., analysis
	// of what happened during the 0th redirect)
	analysisExtDNS(tk, analysis, &info)

	// endpoint (TCP, TLS, HTTP) failure analysis matching with control info (i.e., analysis
	// of what happened during the 0th redirect)
	analysisExtEndpointFailure(tk, analysis, &info)

	// error occurring during redirects (which we can possibly explain if the control
	// succeeded in getting a webpage from the target server)
	analysisExtRedirectErrors(tk, analysis, &info)

	// HTTP success analysis (i.e., only if we manage to get an HTTP response)
	analysisExtHTTPFinalResponse(tk, analysis, &info)

	// handle the cases where the probe and the TH both failed, which we can confidently
	// only evaluate for DNS, TCP, and TLS during the 0-th redirect.
	analysisExtExpectedFailures(tk, analysis, &info)

	// print the content of the analysis only if there's some content to print
	if content := info.String(); content != "" {
		fmt.Printf("\n")
		fmt.Printf("Extended Analysis\n")
		fmt.Printf("-----------------\n")
		fmt.Printf("%s", content)
		fmt.Printf("\n\n")
	}
}

func analysisExtDNS(tk *TestKeys, analysis *minipipeline.WebAnalysis, info io.Writer) {
	// note: here we want to match all the possible conditions because
	// we're processing N >= 1 DNS lookups.

	if failures := analysis.DNSLookupSuccessWithBogonAddresses; failures.Len() > 0 {
		tk.BlockingFlags |= AnalysisBlockingFlagDNSBlocking
		tk.DNSFlags |= AnalysisFlagDNSBogon
		fmt.Fprintf(info, "- transactions with bogon IP addrs: %s\n", failures.String())
	}

	if failures := analysis.DNSLookupUnexpectedFailure; failures.Len() > 0 {
		tk.BlockingFlags |= AnalysisBlockingFlagDNSBlocking
		tk.DNSFlags |= AnalysisDNSFlagUnexpectedFailure
		fmt.Fprintf(info, "- transactions with unexpected DNS lookup failures: %s\n", failures.String())
	}

	if failures := analysis.DNSLookupSuccessWithInvalidAddresses; failures.Len() > 0 {
		tk.BlockingFlags |= AnalysisBlockingFlagDNSBlocking
		tk.DNSFlags |= AnalysisDNSFlagUnexpectedAddrs
		fmt.Fprintf(info, "- transactions with invalid IP addrs: %s\n", failures.String())
	}
}

func analysisExtEndpointFailure(tk *TestKeys, analysis *minipipeline.WebAnalysis, info io.Writer) {
	// note: here we want to match all the possible conditions because
	// we're processing N >= 1 endpoint measurements (with the exception
	// of HTTP but it makes sense to also process HTTP failures here).
	//
	// also note that the definition of "unexpected" implies that we could
	// use the TH to establish some expectations.

	// TCP analysis
	if failures := analysis.TCPConnectUnexpectedFailure; failures.Len() > 0 {
		tk.BlockingFlags |= AnalysisBlockingFlagTCPIPBlocking
		fmt.Fprintf(info, "- transactions with unexpected TCP connect failures: %s\n", failures.String())
	}

	// TLS analysis
	if failures := analysis.TLSHandshakeUnexpectedFailure; failures.Len() > 0 {
		tk.BlockingFlags |= AnalysisBlockingFlagTLSBlocking
		fmt.Fprintf(info, "- transactions with unexpected TLS handshake failures: %s\n", failures.String())
	}

	// HTTP failure analysis
	if failures := analysis.HTTPRoundTripUnexpectedFailure; failures.Len() > 0 {
		tk.BlockingFlags |= AnalysisBlockingFlagHTTPBlocking
		fmt.Fprintf(info, "- transactions with unexpected HTTP round trip failures: %s\n", failures.String())
	}
}

func analysisExtHTTPFinalResponse(tk *TestKeys, analysis *minipipeline.WebAnalysis, info io.Writer) {
	// case #1: HTTP final response without control
	//
	// we don't know what to do in this case.
	if success := analysis.HTTPFinalResponseSuccessTCPWithoutControl; !success.IsNone() {
		fmt.Fprintf(
			info,
			"- there is no control information to compare to the final response (transaction: %d)\n",
			success.Unwrap(),
		)
		return
	}

	// case #2: HTTPS final response without control
	//
	// this is automatic success.
	if success := analysis.HTTPFinalResponseSuccessTLSWithoutControl; !success.IsNone() {
		fmt.Fprintf(info, "- the final response (transaction: %d) uses TLS: automatic success\n", success.Unwrap())
		tk.NullNullFlags |= AnalysisFlagNullNullSuccessfulHTTPS
		tk.BlockingFlags |= AnalysisBlockingFlagSuccess
		return
	}

	// case #3: HTTPS final response with control
	//
	// this is also automatic success.
	if success := analysis.HTTPFinalResponseSuccessTLSWithControl; !success.IsNone() {
		fmt.Fprintf(info, "- the final response (transaction: %d) uses TLS: automatic success\n", success.Unwrap())
		tk.BlockingFlags |= AnalysisBlockingFlagSuccess
		return
	}

	// case #4: HTTP final response with control
	//
	// we need to run HTTPDiff
	if success := analysis.HTTPFinalResponseSuccessTCPWithControl; !success.IsNone() {
		txID := success.Unwrap()
		hds := newAnalysisHTTPDiffStatus(analysis)
		if hds.httpDiff() {
			tk.BlockingFlags |= AnalysisBlockingFlagHTTPDiff
			fmt.Fprintf(info, "- the final response (transaction: %d) differs from the control response\n", txID)
			return
		}
		fmt.Fprintf(info, "- the final response (transaction: %d) matches the control response\n", txID)
		tk.BlockingFlags |= AnalysisBlockingFlagSuccess
		return
	}
}

func analysisExtRedirectErrors(tk *TestKeys, analysis *minipipeline.WebAnalysis, info io.Writer) {
	// Implementation note: we care about cases in which we don't have a final response
	// to compare to and we have unexplained failures. We define "unexplained failure" a
	// failure for which there's no corresponding control information. If we have test
	// helper information telling us that the control server could fetch the final webpage
	// then we can turn these unexplained errors into explained errors.

	switch {
	// case #1: there is a successful final response with or without control
	case !analysis.HTTPFinalResponseSuccessTCPWithoutControl.IsNone():
		return
	case !analysis.HTTPFinalResponseSuccessTLSWithoutControl.IsNone():
		return
	case !analysis.HTTPFinalResponseSuccessTLSWithControl.IsNone():
		return
	case !analysis.HTTPFinalResponseSuccessTCPWithControl.IsNone():
		return

	// case #2: no final response, which is what we care about
	default:
		// fallthrough
	}

	// we care about cases in which the TH succeeded
	if analysis.ControlExpectations.IsNone() {
		return
	}
	expect := analysis.ControlExpectations.Unwrap()
	if expect.FinalResponseFailure.IsNone() {
		return
	}
	if expect.FinalResponseFailure.Unwrap() != "" {
		return
	}

	// okay, now we're in business and we can explain what happened
	//
	// these cases are NOT MUTUALLY EXCLUSIVE because we may have different
	// DNS lookups or endpoints failing in different ways here
	if failures := analysis.DNSLookupUnexplainedFailure; failures.Len() > 0 {
		tk.BlockingFlags |= AnalysisBlockingFlagDNSBlocking
		fmt.Fprintf(
			info, "- transactions with unexplained DNS lookup failures and successful control: %s\n",
			failures.String(),
		)
	}

	if failures := analysis.TCPConnectUnexplainedFailure; failures.Len() > 0 {
		tk.BlockingFlags |= AnalysisBlockingFlagTCPIPBlocking
		fmt.Fprintf(
			info, "- transactions with unexplained TCP connect failures and successful control: %s\n",
			failures.String(),
		)
	}

	if failures := analysis.TLSHandshakeUnexplainedFailure; failures.Len() > 0 {
		tk.BlockingFlags |= AnalysisBlockingFlagTLSBlocking
		fmt.Fprintf(
			info, "- transactions with unexplained TLS handshake failures and successful control: %s\n",
			failures.String(),
		)
	}

	if failures := analysis.HTTPRoundTripUnexplainedFailure; failures.Len() > 0 {
		tk.BlockingFlags |= AnalysisBlockingFlagHTTPBlocking
		fmt.Fprintf(
			info, "- transactions with unexplained HTTP round trip failures and successful control: %s\n",
			failures.String(),
		)
	}
}

func analysisExtExpectedFailures(tk *TestKeys, analysis *minipipeline.WebAnalysis, info io.Writer) {
	// Implementation note: in the "orig" engine this was called the "null-null" analysis
	// because it aimed to address the cases in which failure and accessible were both set
	// to null. We're keeping the original name so we can also keep the same flag we were
	// using before. The flag is a "x_" kind of flag anyway.
	//
	// Also note that these cases ARE NOT MUTUALLY EXCLUSIVE meaning that these conditions
	// could actually happen simultaneously in a bunch of cases.

	if expected := analysis.DNSLookupExpectedFailure; expected.Len() > 0 {
		tk.NullNullFlags |= AnalysisFlagNullNullExpectedDNSLookupFailure
		fmt.Fprintf(
			info, "- transactions with expected DNS lookup failures: %s\n",
			expected.String(),
		)
	}

	if expected := analysis.TCPConnectExpectedFailure; expected.Len() > 0 {
		tk.NullNullFlags |= AnalysisFlagNullNullExpectedTCPConnectFailure
		fmt.Fprintf(
			info, "- transactions with expected TCP connect failures: %s\n",
			expected.String(),
		)
	}

	if expected := analysis.TLSHandshakeExpectedFailure; expected.Len() > 0 {
		tk.NullNullFlags |= AnalysisFlagNullNullExpectedTLSHandshakeFailure
		fmt.Fprintf(
			info, "- transactions with expected TLS handshake failures: %s\n",
			expected.String(),
		)
	}

	// Note: the following flag
	//
	//	tk.NullNullFlags |= AnalysisFlagNullNullSuccessfulHTTPS
	//
	// is set by analysisExtHTTPFinalResponse

	// if the control did not resolve any address but the probe could, this is
	// quite likely censorship injecting addrs for otherwise "down" or nonexisting
	// domains, which lives on as a ghost hunting people
	if !analysis.ControlExpectations.IsNone() {
		expect := analysis.ControlExpectations.Unwrap()
		if expect.DNSAddresses.Len() <= 0 && analysis.DNSLookupSuccess.Len() > 0 {
			tk.NullNullFlags |= AnalysisFlagNullNullUnexpectedDNSLookupSuccess
			fmt.Fprintf(
				info, "- transactions that unexpectedly resolved IP addresses: %s\n",
				analysis.DNSLookupSuccess.String(),
			)
		}
	}
}
