package webconnectivitylte

//
// The extended ("ext") analysis engine.
//
// We analyze all the produced observations without limiting ourselves to
// analyzing observations rooted into getaddrinfo lookups.
//

import (
	"fmt"

	"github.com/ooni/probe-cli/v3/internal/minipipeline"
)

// analysisExtCompute computes the extended analysis.
//
// This function MUTATES the [*TestKeys].
func analysisExtCompute(tk *TestKeys, container *minipipeline.WebObservationsContainer) {
	// compute the web analysis
	analysis := minipipeline.AnalyzeWebObservations(container)

	fmt.Printf("\n")
	fmt.Printf("Extended Analysis\n")
	fmt.Printf("-----------------\n")

	// DNS & address analysis
	if analysis.DNSLookupSuccessWithBogonAddresses.Len() > 0 {
		tk.BlockingFlags |= AnalysisBlockingFlagDNSBlocking
		tk.DNSFlags |= AnalysisFlagDNSBogon
		fmt.Printf(
			"- transactions with bogon IP addresses: %s",
			analysis.DNSLookupSuccessWithBogonAddresses.String(),
		)
	}
	if analysis.DNSLookupUnexpectedFailure.Len() > 0 {
		tk.BlockingFlags |= AnalysisBlockingFlagDNSBlocking
		tk.DNSFlags |= AnalysisDNSFlagUnexpectedFailure
		fmt.Printf(
			"- transactions with unexpected DNS lookup failures: %s",
			analysis.DNSLookupUnexpectedFailure.String(),
		)
	}
	if analysis.DNSLookupSuccessWithInvalidAddresses.Len() > 0 {
		tk.BlockingFlags |= AnalysisBlockingFlagDNSBlocking
		tk.DNSFlags |= AnalysisDNSFlagUnexpectedAddrs
		fmt.Printf(
			"- transactions with invalid IP addrs: %s",
			analysis.DNSLookupSuccessWithInvalidAddresses.String(),
		)
	}

	// TCP analysis
	if analysis.TCPConnectUnexpectedFailure.Len() > 0 {
		tk.BlockingFlags |= AnalysisBlockingFlagTCPIPBlocking
		fmt.Printf(
			"- transactions with unexpected TCP connect failures: %s",
			analysis.TCPConnectUnexpectedFailure.String(),
		)
	}

	// TLS analysis
	if analysis.TLSHandshakeUnexpectedFailure.Len() > 0 {
		tk.BlockingFlags |= AnalysisBlockingFlagTLSBlocking
		fmt.Printf(
			"- transactions with unexpected TLS handshake failures: %s",
			analysis.TLSHandshakeUnexpectedFailure.String(),
		)
	}

	// HTTP analysis
	if analysis.HTTPRoundTripUnexpectedFailure.Len() > 0 {
		tk.BlockingFlags |= AnalysisBlockingFlagHTTPBlocking
		fmt.Printf(
			"- transactions with unexpected HTTP round trip failures: %s",
			analysis.HTTPRoundTripUnexpectedFailure.String(),
		)
	}
	if !analysis.HTTPFinalResponseSuccessTLSWithControl.IsNone() {
		tk.BlockingFlags |= AnalysisBlockingFlagSuccess
		fmt.Printf(
			"- transaction with successful HTTPS response with control: %v",
			analysis.HTTPFinalResponseSuccessTLSWithControl.Unwrap(),
		)
	}
	if !analysis.HTTPFinalResponseSuccessTLSWithoutControl.IsNone() {
		tk.BlockingFlags |= AnalysisBlockingFlagSuccess
		fmt.Printf(
			"- transaction with successful HTTPS response without control: %v",
			analysis.HTTPFinalResponseSuccessTLSWithoutControl.Unwrap(),
		)
	}

	fmt.Printf("\n\n")
}
