package webconnectivity

import "github.com/ooni/probe-cli/v3/internal/model"

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
func (tk *TestKeys) analysisToplevel(logger model.Logger) {
	tk.analysisDNSToplevel(logger)
	tk.analysisTCPIPToplevel()
	tk.analysisHTTPToplevel()
	if (tk.BlockingFlags & analysisBlockingDNS) != 0 {
		tk.Blocking = "dns"
		return
	}
	if (tk.BlockingFlags & analysisBlockingTCPIP) != 0 {
		tk.Blocking = "tcp_ip"
		return
	}
	if (tk.BlockingFlags & analysisBlockingTLSFailure) != 0 {
		tk.Blocking = "http-failure" // backwards compatibility with the spec
		return
	}
	if (tk.BlockingFlags & analysisBlockingHTTPFailure) != 0 {
		tk.Blocking = "http-failure"
		return
	}
	if (tk.BlockingFlags & analysisBlockingHTTPDiff) != 0 {
		tk.Blocking = "http-diff"
		return
	}
	if tk.Accessible == nil || !*tk.Accessible {
		return
	}
	tk.Blocking = false
}
