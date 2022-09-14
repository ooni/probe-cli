package webconnectivity

import (
	"fmt"
	"net"
	"net/url"

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
			"ACCESSIBLE: flags=%d accessible=%+v, blocking=%+v",
			tk.BlockingFlags, tk.Accessible, tk.Blocking,
		)

	default:
		// NullNull remediation
		//
		// If we arrive here, the measurement has failed. However, there are a
		// bunch of cases where we can still explain what happened by applying specific
		// algorithms to detect edge cases.
		//
		// The relative order of these algorithsm matters.

		if tk.analysisNullNullDetectNoAddrs(logger) {
			tk.Blocking = false
			tk.Accessible = false
			logger.Infof(
				"WEBSITE_DOWN_DNS: flags=%d, accessible=%+v, blocking=%+v",
				tk.BlockingFlags, tk.Accessible, tk.Blocking,
			)
			return
		}

		if tk.analysisNullNullDetectAllConnectsFailed(logger) {
			tk.Blocking = false
			tk.Accessible = false
			logger.Infof(
				"WEBSITE_DOWN_TCP: flags=%d, accessible=%+v, blocking=%+v",
				tk.BlockingFlags, tk.Accessible, tk.Blocking,
			)
			return
		}

		if tk.analysisNullNullDetectTLSMisconfigured(logger) {
			tk.Blocking = false
			tk.Accessible = false
			logger.Infof(
				"WEBSITE_DOWN_TLS: flags=%d, accessible=%+v, blocking=%+v",
				tk.BlockingFlags, tk.Accessible, tk.Blocking,
			)
			return
		}

		if tk.analysisNullNullDetectSuccessfulHTTPS(logger) {
			tk.Blocking = false
			tk.Accessible = true
			logger.Infof(
				"ACCESSIBLE_HTTPS: flags=%d, accessible=%+v, blocking=%+v",
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

	// analysisFlagNullNullAllConnectsFailed indicates that all the connect
	// attempts failed both in the probe and in the test helper.
	analysisFlagNullNullAllConnectsFailed

	// analysisFlagNullNullTLSMisconfigured indicates that all the TLS handshake
	// attempts failed both in the probe and in the test helper.
	analysisFlagNullNullTLSMisconfigured

	// analysisFlagNullNullSuccessfulHTTPS indicates that we had no TH data
	// but all the HTTP requests used always HTTPS and never failed.
	analysisFlagNullNullSuccessfulHTTPS
)

// analysisNullNullDetectSuccessfulHTTPS runs when .Blocking = nil and
// .Accessible = nil to flag successul HTTPS measurements chains that
// occurred regardless of whatever else could have gone wrong.
//
// We need all requests to be HTTPS because an HTTP request in the
// chain breaks the ~reasonable assumption that our custom CA bundle
// is enough to protect against MITM. Of course, when we use this
// algorithm, we're not well positioned to flag server-side blocking.
//
// Version 0.4 of the probe implemented a similar algorithm, which
// however ran before other checks. Version, 0.5 on the contrary, runs
// this algorithm if any other heuristics failed.
//
// See https://github.com/ooni/probe/issues/2307 for more info.
func (tk *TestKeys) analysisNullNullDetectSuccessfulHTTPS(logger model.Logger) bool {

	// the chain is sorted from most recent to oldest but it does
	// not matter much since we need to walk all of it.
	//
	// CAVEAT: this code assumes we have a single request chain
	// inside the .Requests field, which seems fine because it's
	// what Web Connectivity should be doing.
	for _, req := range tk.Requests {
		URL, err := url.Parse(req.Request.URL)
		if err != nil {
			// this looks like a bug
			return false
		}
		if URL.Scheme != "https" {
			// the whole chain must be HTTPS
			return false
		}
		if req.Failure != nil {
			// they must all succeed
			return false
		}
		switch req.Response.Code {
		case 200, 301, 302, 307, 308:
		default:
			// the response must be successful or redirect
			return false
		}
	}

	// only if we have at least one request
	if len(tk.Requests) > 0 {
		logger.Info("website likely accessible: seen successful chain of HTTPS transactions")
		tk.NullNullFlags |= analysisFlagNullNullSuccessfulHTTPS
		return true
	}

	// safety net otherwise
	return false
}

// analysisNullNullDetectTLSMisconfigured runs when .Blocking = nil and
// .Accessible = nil to check whether by chance we had TLS issues both on the
// probe side and on the TH side. This problem of detecting misconfiguration
// of the server's TLS stack is discussed at https://github.com/ooni/probe/issues/2300.
func (tk *TestKeys) analysisNullNullDetectTLSMisconfigured(logger model.Logger) bool {
	if tk.Control == nil || tk.Control.TLSHandshake == nil {
		// we need TLS control data to say we are in this case
		return false
	}

	for _, entry := range tk.TLSHandshakes {
		if entry.Failure == nil {
			// we need all attempts to fail to flag this state
			return false
		}
		thEntry, found := tk.Control.TLSHandshake[entry.Address]
		if !found {
			// we need to have seen exactly the same attempts
			return false
		}
		if thEntry.Failure == nil {
			// we need all TH attempts to fail
			return false
		}
		if *entry.Failure != *thEntry.Failure {
			// we need to see the same failure to be sure, which it's
			// possible to do for TLS because we have the same definition
			// of failure rather than being constrained by the legacy
			// implementation of the test helper and Twisted names
			//
			// TODO(bassosimone): this is the obvious algorithm but maybe
			// it's a bit too strict and there is a more lax version of
			// the same algorithm that it's still acceptable?
			return false
		}
	}

	// only if we have had some TLS handshakes for both probe and TH
	if len(tk.TLSHandshakes) > 0 && len(tk.Control.TLSHandshake) > 0 {
		logger.Info("website likely down: all TLS handshake attempts failed for both probe and TH")
		tk.NullNullFlags |= analysisFlagNullNullTLSMisconfigured
		return true
	}

	// safety net in case we've got wrong input
	return false
}

// analysisNullNullDetectAllConnectsFailed attempts to detect whether we are in
// the .Blocking = nil, .Accessible = nil case because all the TCP connect
// attempts by either the probe or the TH have failed.
//
// See https://explorer.ooni.org/measurement/20220911T105037Z_webconnectivity_IT_30722_n1_ruzuQ219SmIO9SrT?input=https://doh.centraleu.pi-dns.com/dns-query?dns=q80BAAABAAAAAAAAA3d3dwdleGFtcGxlA2NvbQAAAQAB
// for an example measurement with this behavior.
//
// See https://github.com/ooni/probe/issues/2299 for the reference issue.
func (tk *TestKeys) analysisNullNullDetectAllConnectsFailed(logger model.Logger) bool {
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
			// we need to have seen exactly the same attempts
			return false
		}
		if thEntry.Failure == nil {
			// we need all TH attempts to fail
			return false
		}
	}

	// only if we have had some addresses to connect
	if len(tk.TCPConnect) > 0 && len(tk.Control.TCPConnect) > 0 {
		logger.Info("website likely down: all TCP connect attempts failed for both probe and TH")
		tk.NullNullFlags |= analysisFlagNullNullAllConnectsFailed
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
	logger.Infof("website likely down: all DNS lookups failed for both probe and TH")
	tk.NullNullFlags |= analysisFlagNullNullNoAddrs
	return true
}
