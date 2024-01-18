package webconnectivitylte

//
// The extended ("ext") analysis engine.
//
// We analyze all the produced observations without limiting ourselves to
// analyzing observations rooted into getaddrinfo lookups.
//

import (
	"fmt"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/minipipeline"
)

// analysisExtCompute computes the extended analysis.
//
// This function MUTATES the [*TestKeys].
func analysisExtCompute(tk *TestKeys, container *minipipeline.WebObservationsContainer) {
	// compute the web analysis
	analysis := minipipeline.AnalyzeWebObservationsWithoutLinearAnalysis(container)

	// prepare for emitting informational messages
	var info strings.Builder

	// DNS & address analysis
	if analysis.DNSLookupSuccessWithBogonAddresses.Len() > 0 {
		tk.BlockingFlags |= AnalysisBlockingFlagDNSBlocking
		tk.DNSFlags |= AnalysisFlagDNSBogon
		fmt.Fprintf(
			&info, "- transactions with bogon IP addresses: %s\n",
			analysis.DNSLookupSuccessWithBogonAddresses.String(),
		)
	}
	if analysis.DNSLookupUnexpectedFailure.Len() > 0 {
		tk.BlockingFlags |= AnalysisBlockingFlagDNSBlocking
		tk.DNSFlags |= AnalysisDNSFlagUnexpectedFailure
		fmt.Fprintf(
			&info, "- transactions with unexpected DNS lookup failures: %s\n",
			analysis.DNSLookupUnexpectedFailure.String(),
		)
	}
	if analysis.DNSLookupSuccessWithInvalidAddresses.Len() > 0 {
		tk.BlockingFlags |= AnalysisBlockingFlagDNSBlocking
		tk.DNSFlags |= AnalysisDNSFlagUnexpectedAddrs
		fmt.Fprintf(
			&info, "- transactions with invalid IP addrs: %s\n",
			analysis.DNSLookupSuccessWithInvalidAddresses.String(),
		)
	}

	// TCP analysis
	if analysis.TCPConnectUnexpectedFailure.Len() > 0 {
		tk.BlockingFlags |= AnalysisBlockingFlagTCPIPBlocking
		fmt.Fprintf(
			&info, "- transactions with unexpected TCP connect failures: %s\n",
			analysis.TCPConnectUnexpectedFailure.String(),
		)
	}

	// TLS analysis
	if analysis.TLSHandshakeUnexpectedFailure.Len() > 0 {
		tk.BlockingFlags |= AnalysisBlockingFlagTLSBlocking
		fmt.Fprintf(
			&info, "- transactions with unexpected TLS handshake failures: %s\n",
			analysis.TLSHandshakeUnexpectedFailure.String(),
		)
	}

	// HTTP failure analysis
	if analysis.HTTPRoundTripUnexpectedFailure.Len() > 0 {
		tk.BlockingFlags |= AnalysisBlockingFlagHTTPBlocking
		fmt.Fprintf(
			&info, "- transactions with unexpected HTTP round trip failures: %s\n",
			analysis.HTTPRoundTripUnexpectedFailure.String(),
		)
	}

	// HTTPS success analysis
	if !analysis.HTTPFinalResponseSuccessTLSWithControl.IsNone() {
		tk.BlockingFlags |= AnalysisBlockingFlagSuccess
		fmt.Fprintf(
			&info, "- transaction with successful HTTPS response with control: %v\n",
			analysis.HTTPFinalResponseSuccessTLSWithControl.Unwrap(),
		)
	}
	if !analysis.HTTPFinalResponseSuccessTLSWithoutControl.IsNone() {
		tk.BlockingFlags |= AnalysisBlockingFlagSuccess
		fmt.Fprintf(
			&info, "- transaction with successful HTTPS response without control: %v\n",
			analysis.HTTPFinalResponseSuccessTLSWithoutControl.Unwrap(),
		)
	}

	// TODO(bassosimone): we need to also compute the HTTPDiff flags here
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
