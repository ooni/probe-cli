package webconnectivityqa

import (
	"github.com/apex/log"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/netemx"
)

// httpDiffWithConsistentDNS verifies the case where there is an HTTP diff
// but the addresses returned by the DNS resolver are consistent.
func httpDiffWithConsistentDNS() *TestCase {
	return &TestCase{
		Name:  "httpDiffWithConsistentDNS",
		Flags: 0,
		Input: "http://www.example.com/",
		Configure: func(env *netemx.QAEnv) {

			// spoof the blockpage
			env.DPIEngine().AddRule(&netem.DPISpoofBlockpageForString{
				HTTPResponse:    netem.DPIFormatHTTPResponse([]byte(netemx.Blockpage)),
				Logger:          log.Log,
				ServerIPAddress: netemx.AddressWwwExampleCom,
				ServerPort:      80,
				String:          "www.example.com",
			})

		},
		ExpectErr: false,
		ExpectTestKeys: &testKeys{
			DNSExperimentFailure:  nil,
			DNSConsistency:        "consistent",
			BodyLengthMatch:       false,
			BodyProportion:        0.12263535551206783,
			StatusCodeMatch:       true,
			HeadersMatch:          false,
			TitleMatch:            false,
			HTTPExperimentFailure: nil,
			XStatus:               64, // StatusAnomalyHTTPDiff
			XDNSFlags:             0,
			XBlockingFlags:        16, // AnalysisBlockingFlagHTTPDiff
			Accessible:            false,
			Blocking:              "http-diff",
		},
	}
}

// httpDiffWithInconsistentDNS verifies the case where there is an HTTP diff
// but the addresses returned by the DNS resolver are inconsistent.
func httpDiffWithInconsistentDNS() *TestCase {
	return &TestCase{
		Name:  "httpDiffWithInconsistentDNS",
		Flags: 0,
		Input: "http://www.example.com/",
		Configure: func(env *netemx.QAEnv) {

			// add DPI rule to force all the cleartext DNS queries to
			// point the client to use the ISPProxyAddress
			env.DPIEngine().AddRule(&netem.DPISpoofDNSResponse{
				Addresses: []string{netemx.ISPProxyAddress},
				Logger:    env.Logger(),
				Domain:    "www.example.com",
			})

			// spoof the blockpage for the legitimate website address as well
			env.DPIEngine().AddRule(&netem.DPISpoofBlockpageForString{
				HTTPResponse:    netem.DPIFormatHTTPResponse([]byte(netemx.Blockpage)),
				Logger:          log.Log,
				ServerIPAddress: netemx.AddressWwwExampleCom,
				ServerPort:      80,
				String:          "www.example.com",
			})

			// spoof the blockpage for the address that we assume the client would use
			env.DPIEngine().AddRule(&netem.DPISpoofBlockpageForString{
				HTTPResponse:    netem.DPIFormatHTTPResponse([]byte(netemx.Blockpage)),
				Logger:          log.Log,
				ServerIPAddress: netemx.ISPProxyAddress,
				ServerPort:      80,
				String:          "www.example.com",
			})

		},
		ExpectErr: false,
		ExpectTestKeys: &testKeys{
			DNSExperimentFailure:  nil,
			DNSConsistency:        "inconsistent",
			HTTPExperimentFailure: nil,
			BodyLengthMatch:       false,
			BodyProportion:        0.12263535551206783,
			StatusCodeMatch:       true,
			HeadersMatch:          false,
			TitleMatch:            false,
			XStatus:               96, // StatusAnomalyHTTPDiff | StatusAnomalyDNS
			XDNSFlags:             4,  // AnalysisDNSFlagUnexpectedAddrs
			XBlockingFlags:        35, // AnalysisBlockingFlagSuccess | AnalysisBlockingFlagDNSBlocking | AnalysisBlockingFlagTCPIPBlocking
			Accessible:            false,
			Blocking:              "dns",
		},
	}
}
