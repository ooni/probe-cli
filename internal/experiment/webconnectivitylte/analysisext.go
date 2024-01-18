package webconnectivitylte

//
// The extended ("ext") analysis engine.
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

	// DNS & address analysis
	analysisExtDNS(tk, analysis, &info)

	// endpoint (TCP, TLS, HTTP) failure analysis
	analysisExtEndpointFailure(tk, analysis, &info)

	// HTTP success analysis
	analysisExtHTTPFinalResponse(tk, analysis, &info)

	// TODO(bassosimone): we need to also compute the null-null flags here

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

	if analysis.DNSLookupSuccessWithBogonAddresses.Len() > 0 {
		tk.BlockingFlags |= AnalysisBlockingFlagDNSBlocking
		tk.DNSFlags |= AnalysisFlagDNSBogon
		fmt.Fprintf(
			info, "- transactions with bogon IP addresses: %s\n",
			analysis.DNSLookupSuccessWithBogonAddresses.String(),
		)
	}

	if analysis.DNSLookupUnexpectedFailure.Len() > 0 {
		tk.BlockingFlags |= AnalysisBlockingFlagDNSBlocking
		tk.DNSFlags |= AnalysisDNSFlagUnexpectedFailure
		fmt.Fprintf(
			info, "- transactions with unexpected DNS lookup failures: %s\n",
			analysis.DNSLookupUnexpectedFailure.String(),
		)
	}

	if analysis.DNSLookupSuccessWithInvalidAddresses.Len() > 0 {
		tk.BlockingFlags |= AnalysisBlockingFlagDNSBlocking
		tk.DNSFlags |= AnalysisDNSFlagUnexpectedAddrs
		fmt.Fprintf(
			info, "- transactions with invalid IP addrs: %s\n",
			analysis.DNSLookupSuccessWithInvalidAddresses.String(),
		)
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
	if analysis.TCPConnectUnexpectedFailure.Len() > 0 {
		tk.BlockingFlags |= AnalysisBlockingFlagTCPIPBlocking
		fmt.Fprintf(
			info, "- transactions with unexpected TCP connect failures: %s\n",
			analysis.TCPConnectUnexpectedFailure.String(),
		)
	}

	// TLS analysis
	if analysis.TLSHandshakeUnexpectedFailure.Len() > 0 {
		tk.BlockingFlags |= AnalysisBlockingFlagTLSBlocking
		fmt.Fprintf(
			info, "- transactions with unexpected TLS handshake failures: %s\n",
			analysis.TLSHandshakeUnexpectedFailure.String(),
		)
	}

	// HTTP failure analysis
	if analysis.HTTPRoundTripUnexpectedFailure.Len() > 0 {
		tk.BlockingFlags |= AnalysisBlockingFlagHTTPBlocking
		fmt.Fprintf(
			info, "- transactions with unexpected HTTP round trip failures: %s\n",
			analysis.HTTPRoundTripUnexpectedFailure.String(),
		)
	}
}

func analysisExtHTTPFinalResponse(tk *TestKeys, analysis *minipipeline.WebAnalysis, info io.Writer) {
	switch {
	// case #1: HTTP final response without control
	//
	// we don't know what to do in this case.
	case !analysis.HTTPFinalResponseSuccessTCPWithoutControl.IsNone():
		txID := analysis.HTTPFinalResponseSuccessTCPWithoutControl.Unwrap()
		fmt.Fprintf(
			info,
			"- there is no control information to compare to the final response (transaction: %d)\n",
			txID,
		)
		return

	// case #2: HTTPS final response without control
	//
	// this is automatic success.
	case !analysis.HTTPFinalResponseSuccessTLSWithoutControl.IsNone():
		txID := analysis.HTTPFinalResponseSuccessTLSWithoutControl.Unwrap()
		fmt.Fprintf(info, "- the final response (transaction: %d) uses TLS: automatic success\n", txID)
		tk.BlockingFlags |= AnalysisBlockingFlagSuccess
		return

	// case #3: HTTPS final response with control
	//
	// this is also automatic success.
	case !analysis.HTTPFinalResponseSuccessTLSWithControl.IsNone():
		txID := analysis.HTTPFinalResponseSuccessTLSWithControl.Unwrap()
		fmt.Fprintf(info, "- the final response (transaction: %d) uses TLS: automatic success\n", txID)
		tk.BlockingFlags |= AnalysisBlockingFlagSuccess
		return

	// case #4: HTTP final response with control
	//
	// we need to run HTTPDiff
	case !analysis.HTTPFinalResponseSuccessTCPWithControl.IsNone():
		txID := analysis.HTTPFinalResponseSuccessTCPWithControl.Unwrap()
		hds := newAnalysisHTTPDiffStatus(analysis)
		if hds.httpDiff() {
			tk.BlockingFlags |= AnalysisBlockingFlagHTTPDiff
			fmt.Fprintf(info, "- the final response (transaction: %d) differs from the control response\n", txID)
			return
		}
		fmt.Fprintf(info, "- the final response (transaction: %d) matches the control response\n", txID)
		tk.BlockingFlags |= AnalysisBlockingFlagSuccess
		return

	// case #5: we don't know
	default:
		return
	}
}
