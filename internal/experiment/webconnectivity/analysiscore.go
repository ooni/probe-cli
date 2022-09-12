package webconnectivity

import (
	"fmt"
	"net"

	"github.com/ooni/probe-cli/v3/internal/model"
)

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
//     | .BlockingFlags                       | .Blocking      | .Accessible |
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
//
// As an improvement over Web Connectivity v0.4, we also attempt to identify
// special subcases of a null, null result to provide the user with more information.
func (tk *TestKeys) analysisToplevel(logger model.Logger) {
	// Since we run after all tasks have completed (or so we assume) we're
	// not going to use any form of locking here.

	// these functions compute the value of XBlockingFlags
	tk.analysisDNSToplevel(logger)
	tk.analysisTCPIPToplevel(logger)
	tk.analysisTLSToplevel(logger)
	tk.analysisHTTPToplevel(logger)

	// now, let's determine .Accessible and .Blocking
	switch {
	case (tk.BlockingFlags & analysisFlagDNSBlocking) != 0:
		tk.Blocking = "dns"
		tk.Accessible = false
		logger.Warnf(
			"ANOMALY: flags=%d accessible=%+v, blocking=%+v",
			tk.BlockingFlags, tk.Accessible, tk.Blocking,
		)

	case (tk.BlockingFlags & analysisFlagTCPIPBlocking) != 0:
		tk.Blocking = "tcp_ip"
		tk.Accessible = false
		logger.Warnf(
			"ANOMALY: flags=%d accessible=%+v, blocking=%+v",
			tk.BlockingFlags, tk.Accessible, tk.Blocking,
		)

	case (tk.BlockingFlags & (analysisFlagTLSBlocking | analysisFlagHTTPBlocking)) != 0:
		tk.Blocking = "http-failure"
		tk.Accessible = false
		logger.Warnf("ANOMALY: flags=%d accessible=%+v, blocking=%+v",
			tk.BlockingFlags, tk.Accessible, tk.Blocking,
		)

	case (tk.BlockingFlags & analysisFlagHTTPDiff) != 0:
		tk.Blocking = "http-diff"
		tk.Accessible = false
		logger.Warnf(
			"ANOMALY: flags=%d accessible=%+v, blocking=%+v",
			tk.BlockingFlags, tk.Accessible, tk.Blocking,
		)

	case tk.BlockingFlags == analysisFlagSuccess:
		tk.Blocking = false
		tk.Accessible = true
		logger.Infof(
			"SUCCESS: flags=%d accessible=%+v, blocking=%+v",
			tk.BlockingFlags, tk.Accessible, tk.Blocking,
		)

	default:
		if tk.analysisNullNullDetectNoAddrs(logger) {
			tk.Blocking = false
			tk.Accessible = false
			logger.Infof(
				"NO_AVAILABLE_ADDRS: flags=%d, accessible=%+v, blocking=%+v",
				tk.BlockingFlags, tk.Accessible, tk.Blocking,
			)
			return
		}
		if tk.analysisFlagNullNullDetectAllConnectFailed(logger) {
			tk.Blocking = false
			tk.Accessible = false
			logger.Infof(
				"ALL_CONNECTS_FAILED: flags=%d, accessible=%+v, blocking=%+v",
				tk.BlockingFlags, tk.Accessible, tk.Blocking,
			)
			return
		}
		tk.Blocking = nil
		tk.Accessible = nil
		logger.Warnf(
			"UNKNOWN: flags=%d, accessible=%+v, blocking=%+v",
			tk.BlockingFlags, tk.Accessible, tk.Blocking,
		)
	}
}

const (
	// analysisFlagNullNullNoAddrs indicates neither the probe nor the TH were
	// able to get any IP addresses from any resolver.
	analysisFlagNullNullNoAddrs = 1 << iota

	// analysisFlagNullNullAllConnectFailed indicates that all the connect
	// attempts failed both in the probe and in the test helper.
	analysisFlagNullNullAllConnectFailed
)

// analysisNullMullDetectAllConnectFailed attempts to detect whether we are in
// the .Blocking = nil, .Accessible = nil case because all the TCP connect
// attempts by either the probe or the TH have failed.
//
// See https://explorer.ooni.org/measurement/20220911T105037Z_webconnectivity_IT_30722_n1_ruzuQ219SmIO9SrT?input=https://doh.centraleu.pi-dns.com/dns-query?dns=q80BAAABAAAAAAAAA3d3dwdleGFtcGxlA2NvbQAAAQAB
// for an example measurement with this behavior.
func (tk *TestKeys) analysisFlagNullNullDetectAllConnectFailed(logger model.Logger) bool {
	if tk.Control == nil {
		// we need control data to say we're in this case
		return false
	}

	for _, entry := range tk.TCPConnect {
		if entry.Status.Failure == nil {
			// we need all connect attempts to fail
			return false
		}
		epnt := net.JoinHostPort(entry.IP, fmt.Sprintf("%d", entry.Port))
		thEntry, found := tk.Control.TCPConnect[epnt]
		if !found {
			// we need exactly the same attempts to have failed
			return false
		}
		if thEntry.Failure == nil {
			// we need all TH attempts to fail
			return false
		}
	}

	// only if we have had some addresses to connect
	if len(tk.TCPConnect) > 0 && len(tk.Control.TCPConnect) > 0 {
		logger.Info("All TCP connect attempts failed for both probe and TH")
		tk.NullNullFlags |= analysisFlagNullNullAllConnectFailed
		return true
	}

	// safety net in case we're passed empty lists/maps
	return false
}

// analysisNullNullDetectNoAddrs attempts to see whether we
// ended up into the .Blocking = nil, .Accessible = nil case because
// the domain is expired and all queries returned no addresses.
//
// See https://github.com/ooni/probe/issues/2290 for further
// documentation about the issue we're solving here.
//
// It would be tempting to check specifically for NXDOMAIN here, but we
// know it is problematic do that. In fact, on Android the getaddrinfo
// resolver always returns EAI_NODATA on error, regardless of the actual
// error that may have occurred in the Android DNS backend.
//
// See https://github.com/ooni/probe/issues/2029 for more information
// on Android's getaddrinfo behavior.
func (tk *TestKeys) analysisNullNullDetectNoAddrs(logger model.Logger) bool {
	if tk.Control == nil {
		// we need control data to say we're in this case
		return false
	}
	for _, query := range tk.Queries {
		if len(query.Answers) > 0 {
			// when a query has answers, we're not in the NoAddresses case
			return false
		}
	}
	if len(tk.TCPConnect) > 0 {
		// if we attempted TCP connect, we're not in the NoAddresses case
		return false
	}
	if len(tk.TLSHandshakes) > 0 {
		// if we attempted TLS handshakes, we're not in the NoAddresses case
		return false
	}
	if len(tk.Control.DNS.Addrs) > 0 {
		// when the TH resolved addresses, we're not in the NoAddresses case
		return false
	}
	if len(tk.Control.TCPConnect) > 0 {
		// when the TH used addresses, we're not in the NoAddresses case
		return false
	}
	logger.Infof("Neither the probe nor the TH resolved any addresses")
	tk.NullNullFlags |= analysisFlagNullNullNoAddrs
	return true
}
