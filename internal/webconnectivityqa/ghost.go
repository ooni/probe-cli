package webconnectivityqa

import (
	"github.com/apex/log"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/netemx"
)

// ghostDNSBlockingWithHTTP is the case where the domain does not exist anymore but
// there's still ghost censorship because of the censor DNS censoring configuration, which
// says that we should censor the domain by returning a specific IP address.
//
// See https://github.com/ooni/probe/issues/2308.
func ghostDNSBlockingWithHTTP() *TestCase {
	return &TestCase{
		Name:  "ghostDNSBlockingWithHTTP",
		Flags: TestCaseFlagNoV04,
		Input: "http://itsat.info/",
		Configure: func(env *netemx.QAEnv) {
			// remove the record so that the DNS query returns NXDOMAIN
			env.ISPResolverConfig().RemoveRecord("itsat.info")
			env.OtherResolversConfig().RemoveRecord("itsat.info")

			// however introduce a rule causing DNS to respond to the query
			env.DPIEngine().AddRule(&netem.DPISpoofDNSResponse{
				Addresses: []string{
					netemx.AddressPublicBlockpage,
				},
				Logger: log.Log,
				Domain: "itsat.info",
			})
		},
		ExpectErr: false,
		ExpectTestKeys: &TestKeys{
			DNSExperimentFailure: nil,
			DNSConsistency:       "inconsistent",
			XBlockingFlags:       16, // AnalysisBlockingFlagHTTPDiff
			XNullNullFlags:       18, // AnalysisFlagNullNullExpectedTCPConnectFailure | AnalysisFlagNullNullUnexpectedDNSLookupSuccess
			XStatus:              16, // StatusAnomalyControlFailure
			Accessible:           false,
			Blocking:             "dns",
		},
	}
}

// ghostDNSBlockingWithHTTPS is the case where the domain does not exist anymore but
// there's still ghost censorship because of the censor DNS censoring configuration, which
// says that we should censor the domain by returning a specific IP address.
//
// See https://github.com/ooni/probe/issues/2308.
func ghostDNSBlockingWithHTTPS() *TestCase {
	return &TestCase{
		Name:  "ghostDNSBlockingWithHTTPS",
		Flags: 0,
		Input: "https://itsat.info/",
		Configure: func(env *netemx.QAEnv) {
			// remove the record so that the DNS query returns NXDOMAIN
			env.ISPResolverConfig().RemoveRecord("itsat.info")
			env.OtherResolversConfig().RemoveRecord("itsat.info")

			// however introduce a rule causing DNS to respond to the query
			env.DPIEngine().AddRule(&netem.DPISpoofDNSResponse{
				Addresses: []string{
					netemx.AddressPublicBlockpage,
				},
				Logger: log.Log,
				Domain: "itsat.info",
			})
		},
		ExpectErr: false,
		ExpectTestKeys: &TestKeys{
			DNSExperimentFailure:  nil,
			DNSConsistency:        "inconsistent",
			HTTPExperimentFailure: "connection_refused",
			XNullNullFlags:        18,   // AnalysisFlagNullNullExpectedTCPConnectFailure | AnalysisFlagNullNullUnexpectedDNSLookupSuccess
			XStatus:               4256, // StatusExperimentConnect | StatusAnomalyDNS | StatusAnomalyConnect
			Accessible:            false,
			Blocking:              "dns",
		},
	}
}
