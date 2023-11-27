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
			XBlockingFlags:        16, // analysisFlagHTTPDiff
			Accessible:            false,
			Blocking:              "http-diff",
		},
	}
}

// httpDiffWithInconsistentDNS verifies the case where there is an HTTP diff
// but the addresses returned by the DNS resolver are inconsistent.
func httpDiffWithInconsistentDNS() *TestCase {
	return &TestCase{
		Name: "httpDiffWithInconsistentDNS",
		// With v0.5 we conclude that the DNS is consistent because we can still
		// perform TLS connections with the given addresses. Disable v0.4 because
		// it does not reach to the same conclusion.
		//
		// TODO(bassosimone): maybe we should create another test case where
		// we end up with having a truly inconsistent DNS.
		Flags: TestCaseFlagNoV04,
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
			DNSConsistency:        "consistent",
			HTTPExperimentFailure: nil,
			BodyLengthMatch:       false,
			BodyProportion:        0.12263535551206783,
			StatusCodeMatch:       true,
			HeadersMatch:          false,
			TitleMatch:            false,
			XStatus:               96, // StatusAnomalyHTTPDiff | StatusAnomalyDNS
			XDNSFlags:             0,
			XBlockingFlags:        16, // analysisFlagHTTPDiff
			Accessible:            false,
			Blocking:              "http-diff",
		},
	}
}
