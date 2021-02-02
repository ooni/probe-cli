package webconnectivity

import (
	"strings"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/webconnectivity/internal"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/errorx"
)

// The following set of status flags identifies in a more nuanced way the
// reason why we say something is blocked, accessible, etc.
//
// This is an experimental implementation. The objective is to start using
// it and learning from it, to eventually understand in which direction
// the Web Connectivity experiment should evolve. For example, there are
// a bunch of flags where our understandind is fuzzy or unclear.
//
// It also helps to write more precise unit and integration tests.
const (
	StatusSuccessSecure    = 1 << iota // success when using HTTPS
	StatusSuccessCleartext             // success when using HTTP
	StatusSuccessNXDOMAIN              // probe and control agree on NXDOMAIN

	StatusAnomalyControlUnreachable // cannot access the control
	StatusAnomalyControlFailure     // control failed for HTTP
	StatusAnomalyDNS                // probe seems blocked by their DNS
	StatusAnomalyHTTPDiff           // probe and control do not agree on HTTP features
	StatusAnomalyConnect            // we saw an error when connecting
	StatusAnomalyReadWrite          // we saw an error when doing I/O
	StatusAnomalyUnknown            // we don't know when the error happened
	StatusAnomalyTLSHandshake       // we think error was during TLS handshake

	StatusExperimentDNS     // we noticed something in the DNS experiment
	StatusExperimentConnect // ... in the connect experiment
	StatusExperimentHTTP    // ... in the HTTP experiment

	StatusBugNoRequests // this should never happen
)

// Summary contains the Web Connectivity summary.
type Summary struct {
	// Accessible is nil when the measurement failed, true if we do
	// not think there was blocking, false in case of blocking.
	Accessible *bool `json:"accessible"`

	// BlockingReason indicates the cause of blocking when the Accessible
	// variable is false. BlockingReason is meaningless otherwise.
	//
	// This is an intermediate variable used to compute Blocking, which
	// is what OONI data consumers expect to see.
	BlockingReason *string `json:"-"`

	// Blocking implements the blocking variable as expected by OONI
	// data consumers. See DetermineBlocking's docs.
	Blocking interface{} `json:"blocking"`

	// Status contains zero or more status flags. This is currently
	// an experimental interface subject to change at any time.
	Status int64 `json:"x_status"`
}

// DetermineBlocking returns the value of Summary.Blocking according to
// the expectations of OONI data consumers (nil|false|string).
//
// Measurement Kit sets blocking to false when accessible is true. The spec
// doesn't mention this possibility, as of 2019-08-20-001. Yet we implemented
// it back in 2016, with little explanation <https://git.io/JJHOl>.
//
// We eventually managed to link such a change with the 0.3.4 release of
// Measurement Kit <https://git.io/JJHOS>. This led us to find out the
// related issue #867 <https://git.io/JJHOH>. From this issue it become
// clear that the change on Measurement Kit was applied to mirror a change
// implemented in OONI Probe Legacy <https://git.io/JJH3T>. In such a
// change, determine_blocking() was modified to return False in case no
// blocking was detected, to distinguish this case from the case where
// there was an early failure in the experiment.
//
// Indeed, the OONI Android app uses the case where `blocking` is `null`
// to flag failed tests. Instead, success is identified by `blocking` being
// false and all other cases indicate anomaly <https://git.io/JJH3C>.
//
// Because of that, we must preserve the original behaviour.
func DetermineBlocking(s Summary) interface{} {
	if s.Accessible != nil && *s.Accessible == true {
		return false
	}
	return s.BlockingReason
}

// Log logs the summary using the provided logger.
func (s Summary) Log(logger model.Logger) {
	logger.Infof("Blocking: %+v", internal.StringPointerToString(s.BlockingReason))
	logger.Infof("Accessible: %+v", internal.BoolPointerToString(s.Accessible))
}

// Summarize computes the summary from the TestKeys.
func Summarize(tk *TestKeys) (out Summary) {
	// Make sure we correctly set out.Blocking's value.
	defer func() {
		out.Blocking = DetermineBlocking(out)
	}()
	var (
		accessible   = true
		inaccessible = false
		dns          = "dns"
		httpDiff     = "http-diff"
		httpFailure  = "http-failure"
		tcpIP        = "tcp_ip"
	)
	// If the measurement was for an HTTPS website and the HTTP experiment
	// succeded, then either there is a compromised CA in our pool (which is
	// certifi-go), or there is transparent proxying, or we are actually
	// speaking with the legit server. We assume the latter. This applies
	// also to cases in which we are redirected to HTTPS.
	if len(tk.Requests) > 0 && tk.Requests[0].Failure == nil &&
		strings.HasPrefix(tk.Requests[0].Request.URL, "https://") {
		out.Accessible = &accessible
		out.Status |= StatusSuccessSecure
		return
	}
	// If we couldn't contact the control, we cannot do much more here.
	if tk.ControlFailure != nil {
		out.Status |= StatusAnomalyControlUnreachable
		return
	}
	// If DNS failed with NXDOMAIN and the control DNS is consistent, then it
	// means this website does not exist anymore.
	if tk.DNSExperimentFailure != nil &&
		*tk.DNSExperimentFailure == errorx.FailureDNSNXDOMAINError &&
		tk.DNSConsistency != nil && *tk.DNSConsistency == DNSConsistent {
		// TODO(bassosimone): MK flags this as accessible. This result is debateable. We
		// are doing what MK does. But we most likely want to make it better later.
		//
		// See <https://github.com/ooni/probe-cli/v3/internal/engine/issues/579>.
		out.Accessible = &accessible
		out.Status |= StatusSuccessNXDOMAIN | StatusExperimentDNS
		return
	}
	// Otherwise, if DNS failed with NXDOMAIN, it's DNS based blocking.
	// TODO(bassosimone): do we wanna include other errors here? Like timeout?
	if tk.DNSExperimentFailure != nil &&
		*tk.DNSExperimentFailure == errorx.FailureDNSNXDOMAINError {
		out.Accessible = &inaccessible
		out.BlockingReason = &dns
		out.Status |= StatusAnomalyDNS | StatusExperimentDNS
		return
	}
	// If we tried to connect more than once and never succeded and we were
	// able to measure DNS consistency, then we can conclude something.
	if tk.TCPConnectAttempts > 0 && tk.TCPConnectSuccesses <= 0 && tk.DNSConsistency != nil {
		out.Status |= StatusAnomalyConnect | StatusExperimentConnect
		switch *tk.DNSConsistency {
		case DNSConsistent:
			// If the DNS is consistent, then it's TCP/IP blocking.
			out.BlockingReason = &tcpIP
			out.Accessible = &inaccessible
		case DNSInconsistent:
			// Otherwise, the culprit is the DNS.
			out.BlockingReason = &dns
			out.Accessible = &inaccessible
			out.Status |= StatusAnomalyDNS
		default:
			// this case should not happen with this implementation
			// so it's fine to leave this as unknown
			out.Status |= StatusAnomalyUnknown
		}
		return
	}
	// If the control failed for HTTP it's not immediate for us to
	// say anything specific on this measurement.
	if tk.Control.HTTPRequest.Failure != nil {
		out.Status |= StatusAnomalyControlFailure
		return
	}
	// Likewise, if we don't have requests to examine, leave it.
	if len(tk.Requests) < 1 {
		out.Status |= StatusBugNoRequests
		return
	}
	// If the HTTP measurement failed there could be a bunch of reasons
	// why this occurred, because of HTTP redirects. Try to guess what
	// could have been wrong by inspecting the error code.
	if tk.Requests[0].Failure != nil {
		out.Status |= StatusExperimentHTTP
		switch *tk.Requests[0].Failure {
		case errorx.FailureConnectionRefused:
			// This is possibly because a subsequent connection to some
			// other endpoint has been blocked. We call this http-failure
			// because this is what MK would actually do.
			out.BlockingReason = &httpFailure
			out.Accessible = &inaccessible
			out.Status |= StatusAnomalyConnect
		case errorx.FailureConnectionReset:
			// We don't currently support TLS failures and we don't have a
			// way to know if it was during TLS or later. So, for now we are
			// going to call this error condition an http-failure.
			out.BlockingReason = &httpFailure
			out.Accessible = &inaccessible
			out.Status |= StatusAnomalyReadWrite
		case errorx.FailureDNSNXDOMAINError:
			// This is possibly because a subsequent resolution to
			// some other domain name has been blocked.
			out.BlockingReason = &dns
			out.Accessible = &inaccessible
			out.Status |= StatusAnomalyDNS
		case errorx.FailureEOFError:
			// We have seen this happening with TLS handshakes as well as
			// sometimes with HTTP blocking. So http-failure.
			out.BlockingReason = &httpFailure
			out.Accessible = &inaccessible
			out.Status |= StatusAnomalyReadWrite
		case errorx.FailureGenericTimeoutError:
			// Alas, here we don't know whether it's connect or whether it's
			// perhaps the TLS handshake. So use the same classification used by MK.
			out.BlockingReason = &httpFailure
			out.Accessible = &inaccessible
			out.Status |= StatusAnomalyUnknown
		case errorx.FailureSSLInvalidHostname,
			errorx.FailureSSLInvalidCertificate,
			errorx.FailureSSLUnknownAuthority:
			// We treat these three cases equally. Misconfiguration is a bit
			// less likely since we also checked with the control. Since there
			// is no TLS, for now we're going to call this http-failure.
			out.BlockingReason = &httpFailure
			out.Accessible = &inaccessible
			out.Status |= StatusAnomalyTLSHandshake
		default:
			// We have not been able to classify the error. Could this perhaps be
			// caused by a programmer's error? Let us be conservative.
		}
		// So, good that we have classified the error. Yet, how long is the
		// redirect chain? If it's exactly one and we have determined that we
		// should not trust the resolver, then let's bet on the DNS. If the
		// chain is longer, for now better to be conservative. (I would argue
		// that with a lying DNS that's likely the culprit, honestly.)
		if out.BlockingReason != nil && len(tk.Requests) == 1 &&
			tk.DNSConsistency != nil && *tk.DNSConsistency == DNSInconsistent {
			out.BlockingReason = &dns
			out.Status |= StatusAnomalyDNS
		}
		return
	}
	// So the HTTP request did not fail in the measurement and did not
	// fail in the control as well, didn't it? Then, let us try to guess
	// whether we've got the expected webpage after all. This set of
	// conditions is adapted from MK v0.10.11.
	if tk.StatusCodeMatch != nil && *tk.StatusCodeMatch {
		if tk.BodyLengthMatch != nil && *tk.BodyLengthMatch {
			out.Accessible = &accessible
			out.Status |= StatusSuccessCleartext
			return
		}
		if tk.HeadersMatch != nil && *tk.HeadersMatch {
			out.Accessible = &accessible
			out.Status |= StatusSuccessCleartext
			return
		}
		if tk.TitleMatch != nil && *tk.TitleMatch {
			out.Accessible = &accessible
			out.Status |= StatusSuccessCleartext
			return
		}
	}
	// Set the status flag first
	out.Status |= StatusAnomalyHTTPDiff
	// It seems we didn't get the expected web page. What now? Well, if
	// the DNS does not seem trustworthy, let us blame it.
	if tk.DNSConsistency != nil && *tk.DNSConsistency == DNSInconsistent {
		out.BlockingReason = &dns
		out.Accessible = &inaccessible
		out.Status |= StatusAnomalyDNS
		return
	}
	// The only remaining conclusion seems that the web page we have got
	// doesn't match what we were expecting.
	out.BlockingReason = &httpDiff
	out.Accessible = &inaccessible
	return
}
