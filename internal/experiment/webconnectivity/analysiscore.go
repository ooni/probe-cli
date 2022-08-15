package webconnectivity

//
// Core analysis
//

const (
	// analysisBlockingDNS indicates there's blocking at the DNS level.
	analysisBlockingDNS = 1 << iota

	// analysisBlockingTCPIP indicates there's blocking at the TCP/IP level.
	analysisBlockingTCPIP

	// analysisBlockingTLSFailure indicates there were TLS issues.
	analysisBlockingTLSFailure

	// analysisBlockingHTTPFailure indicates there was an HTTP failure.
	analysisBlockingHTTPFailure

	// analysisBlockingHTTPDiff indicates there's an HTTP diff.
	analysisBlockingHTTPDiff
)

// analysisToplevel is the toplevel function that analyses the results
// of the experiment once all network tasks have completed.
func (tk *TestKeys) analysisToplevel() {
	tk.analysisDNSToplevel()
	tk.analysisTCPIPToplevel()
	tk.analysisHTTPToplevel()
}
