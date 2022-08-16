package webconnectivity

import "github.com/ooni/probe-cli/v3/internal/model"

//
// Core analysis
//

// These flags determine the context of TestKeys.Blocking. However, while .Blocking
// is an enumeration, these flags allow to describe multiple blocking methods.
const (
	// analysisFlagDNSBlocking indicates there's blocking at the DNS level.
	analysisFlagDNSBlocking = 1 << iota

	// analysisFlagTCPIPBlocking indicates there's blocking at the TCP/IP level.
	analysisFlagTCPIPBlocking

	// analysisFlagTLSBlocking indicates there were TLS issues.
	analysisFlagTLSBlocking

	// analysisFlagHTTPBlocking indicates there was an HTTP failure.
	analysisFlagHTTPBlocking

	// analysisFlagHTTPDiff indicates there's an HTTP diff.
	analysisFlagHTTPDiff

	// analysisFlagSuccess indicates we did not detect any blocking.
	analysisFlagSuccess
)

// analysisToplevel is the toplevel function that analyses the results
// of the experiment once all network tasks have completed.
//
// The ultimate objective of this function is to set the toplevel flags
// used by the backend to score results. These flags are:
//
// - blocking (and x_blocking_flags) which contain information about
// the detected blocking method (or methods);
//
// - accessible which contains information on whether we think we
// could access the resource somehow.
//
// Originally, Web Connectivity only had a blocking scalar value so
// we could see ourselves in one of the following cases:
//
//     +----------+------------+--------------------------+
//     | Blocking | Accessible | Meaning                  |
//     +----------+------------+--------------------------+
//     | null     | null       | Probe analysis error     |
//     +----------+------------+--------------------------+
//     | false    | true       | We detected no blocking  |
//     +----------+------------+--------------------------+
//     | "..."    | false      | We detected blocking     |
//     +----------+------------+--------------------------+
//
// While it would be possible in this implementation, which has a granular
// definition of blocking (x_blocking_flags), to set accessible to mean
// whether we could access the resource in some conditions, it seems quite
// dangerous to deviate from the original behavior.
//
// Our code will NEVER set .Blocking or .Accessible outside of this function
// and we'll instead rely on XBlockingFlags. This function's job is to call
// other functions that compute the .XBlockingFlags and then to assign the value
// of .Blocking and .Accessible from the .XBlockingFlags value.
//
// Accordingly, this is how we map the value of the .XBlockingFlags to the
// values of .Blocking and .Accessible:
//
//     +--------------------------------------+----------------+-------------+
//     | XBlockingFlags                       | .Blocking      | .Accessible |
//     +--------------------------------------+----------------+-------------+
//     | (& DNSBlocking) != 0                 | "dns"          | false       |
//     +--------------------------------------+----------------+-------------+
//     | (& TCPIPBlocking) != 0               | "tcp_ip"       | false       |
//     +--------------------------------------+----------------+-------------+
//     | (& (TLSBlocking|HTTPBlocking)) != 0  | "http-failure" | false       |
//     +--------------------------------------+----------------+-------------+
//     | (& HTTPDiff) != 0                    | "http-diff"    | false       |
//     +--------------------------------------+----------------+-------------+
//     | == FlagSuccess                       | false          | true        |
//     +--------------------------------------+----------------+-------------+
//     | otherwise                            | null           | null        |
//     +--------------------------------------+----------------+-------------+
//
// It's a very simple rule, that should preserve previous semantics.
func (tk *TestKeys) analysisToplevel(logger model.Logger) {
	// Since we run after all tasks have completed (or so we assume) we're
	// not going to use any form of locking here.

	// these functions compute the value of XBlockingFlags
	tk.analysisDNSToplevel(logger)
	tk.analysisTCPIPToplevel(logger)
	tk.analysisHTTPToplevel(logger)

	// now, let's determine .Accessible and .Blocking
	switch {
	case (tk.XBlockingFlags & analysisFlagDNSBlocking) != 0:
		tk.Blocking = "dns"
		tk.Accessible = false
		logger.Warnf(
			"ANOMALY: flags=%d accessible=%+v, blocking=%+v",
			tk.XBlockingFlags, tk.Accessible, tk.Blocking,
		)

	case (tk.XBlockingFlags & analysisFlagTCPIPBlocking) != 0:
		tk.Blocking = "tcp_ip"
		tk.Accessible = false
		logger.Warnf(
			"ANOMALY: flags=%d accessible=%+v, blocking=%+v",
			tk.XBlockingFlags, tk.Accessible, tk.Blocking,
		)

	case (tk.XBlockingFlags & (analysisFlagTLSBlocking | analysisFlagHTTPBlocking)) != 0:
		tk.Blocking = "http-failure"
		tk.Accessible = false
		logger.Warnf("ANOMALY: flags=%d accessible=%+v, blocking=%+v",
			tk.XBlockingFlags, tk.Accessible, tk.Blocking,
		)

	case (tk.XBlockingFlags & analysisFlagHTTPDiff) != 0:
		tk.Blocking = "http-diff"
		tk.Accessible = false
		logger.Warnf(
			"ANOMALY: flags=%d accessible=%+v, blocking=%+v",
			tk.XBlockingFlags, tk.Accessible, tk.Blocking,
		)

	case tk.XBlockingFlags == analysisFlagSuccess:
		tk.Blocking = false
		tk.Accessible = true
		logger.Infof(
			"SUCCESS: flags=%d accessible=%+v, blocking=%+v",
			tk.XBlockingFlags, tk.Accessible, tk.Blocking,
		)

	default:
		tk.Blocking = nil
		tk.Accessible = nil
		logger.Warnf(
			"UNKNOWN: flags=%d, accessible=%+v, blocking=%+v",
			tk.XBlockingFlags, tk.Accessible, tk.Blocking,
		)
	}
}
