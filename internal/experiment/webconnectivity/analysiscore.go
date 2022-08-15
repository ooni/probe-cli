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
	tk.analysisTCPIPToplevel(logger)
	tk.analysisHTTPToplevel(logger)

	accessibleFalse := false

	if (tk.BlockingFlags & analysisBlockingDNS) != 0 {
		logger.Warnf("BLOCKING: dns => NOT ACCESSIBLE")
		tk.Blocking = "dns"
		tk.Accessible = &accessibleFalse
		return
	}

	if (tk.BlockingFlags & analysisBlockingTCPIP) != 0 {
		logger.Warnf("BLOCKING: tcp_ip => NOT ACCESSIBLE")
		tk.Blocking = "tcp_ip"
		tk.Accessible = &accessibleFalse
		return
	}

	if (tk.BlockingFlags & analysisBlockingTLSFailure) != 0 {
		logger.Warnf("BLOCKING: http-failure (TLS) => NOT ACCESSIBLE")
		tk.Blocking = "http-failure" // backwards compatibility with the spec
		tk.Accessible = &accessibleFalse
		return
	}

	if (tk.BlockingFlags & analysisBlockingHTTPFailure) != 0 {
		logger.Warnf("BLOCKING: http-failure (HTTP) => NOT ACCESSIBLE")
		tk.Blocking = "http-failure"
		tk.Accessible = &accessibleFalse
		return
	}

	if (tk.BlockingFlags & analysisBlockingHTTPDiff) != 0 {
		logger.Warnf("BLOCKING: http-diff => NOT ACCESSIBLE")
		tk.Blocking = "http-diff"
		tk.Accessible = &accessibleFalse
		return
	}

	if tk.Accessible == nil {
		logger.Warnf("ACCESSIBLE: null")
		return
	}

	if !*tk.Accessible {
		logger.Warnf("ACCESSIBLE: false") // can this happen?
		return
	}

	logger.Infof("BLOCKING: false")
	logger.Infof("ACCESSIBLE: true")
	tk.Blocking = false
}
