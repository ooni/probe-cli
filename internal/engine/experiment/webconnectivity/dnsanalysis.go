package webconnectivity

import (
	"net"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// DNSAnalysisResult contains the results of analysing comparing
// the measurement and the control DNS results.
type DNSAnalysisResult struct {
	DNSConsistency *string `json:"dns_consistency"`
}

// DNSNameError is the error returned by the control on NXDOMAIN
const DNSNameError = model.THDNSNameError

var (
	// DNSConsistent indicates that the measurement and the
	// control have consistent DNS results.
	DNSConsistent = "consistent"

	// DNSInconsistent indicates that the measurement and the
	// control have inconsistent DNS results.
	DNSInconsistent = "inconsistent"
)

// DNSAnalysis compares the measurement and the control DNS results. This
// implementation is a simplified version of the implementation of the same
// check implemented in Measurement Kit v0.10.11.
func DNSAnalysis(URL *url.URL, measurement DNSLookupResult,
	control ControlResponse) (out DNSAnalysisResult) {
	// 0. start assuming it's not consistent
	out.DNSConsistency = &DNSInconsistent
	// 1. flip to consistent if we're targeting an IP address because the
	// control will actually return dns_name_error in this case.
	if net.ParseIP(URL.Hostname()) != nil {
		out.DNSConsistency = &DNSConsistent
		return
	}
	// 2. flip to consistent if the failures are compatible
	if measurement.Failure != nil && control.DNS.Failure != nil {
		switch *control.DNS.Failure {
		case DNSNameError: // the control returns this on NXDOMAIN error
			switch *measurement.Failure {
			// When the Android getaddrinfo cache says "no data" (meaning basically
			// "I don't know, mate") _and_ the test helper says NXDOMAIN, we can
			// be ~confident that there's also NXDOMAIN on the Android side.
			//
			// See also https://github.com/ooni/probe/issues/2029.
			case netxlite.FailureDNSNXDOMAINError,
				netxlite.FailureAndroidDNSCacheNoData:
				out.DNSConsistency = &DNSConsistent
			}
		}
		return
	}
	// 3. flip to consistent if measurement and control returned IP addresses
	// that belong to the same Autonomous System(s).
	//
	// This specific check is present in MK's implementation.
	//
	// Note that this covers also the cases where the measurement contains only
	// bogons while the control does not contain bogons.
	//
	// Note that this also covers the cases where results are equal.
	const (
		inMeasurement = 1 << 0
		inControl     = 1 << 1
		inBoth        = inMeasurement | inControl
	)
	asnmap := make(map[int64]int)
	for _, asn := range measurement.Addrs {
		asnmap[asn] |= inMeasurement
	}
	for _, asn := range control.DNS.ASNs {
		asnmap[asn] |= inControl
	}
	for key, value := range asnmap {
		// zero means that ASN lookup failed
		if key != 0 && (value&inBoth) == inBoth {
			out.DNSConsistency = &DNSConsistent
			return
		}
	}
	// 4. when ASN lookup failed (unlikely), check whether
	// there is overlap in the returned IP addresses
	ipmap := make(map[string]int)
	for ip := range measurement.Addrs {
		ipmap[ip] |= inMeasurement
	}
	for _, ip := range control.DNS.Addrs {
		ipmap[ip] |= inControl
	}
	for key, value := range ipmap {
		// just in case an empty string slipped through
		if key != "" && (value&inBoth) == inBoth {
			out.DNSConsistency = &DNSConsistent
			return
		}
	}
	// 5. conclude that measurement and control are inconsistent
	return
}
