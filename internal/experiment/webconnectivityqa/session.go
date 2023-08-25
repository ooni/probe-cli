package webconnectivityqa

import (
	"errors"
	"strconv"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// nwSession creates a new [model.ExperimentSession].
func newSession(client model.HTTPClient, logger model.Logger) model.ExperimentSession {
	return &mocks.Session{
		MockGetTestHelpersByName: func(name string) ([]model.OOAPIService, bool) {
			output := []model.OOAPIService{{
				Address: "https://0.th.ooni.org/",
				Type:    "https",
				Front:   "",
			}, {
				Address: "https://1.th.ooni.org/",
				Type:    "https",
				Front:   "",
			}, {
				Address: "https://2.th.ooni.org/",
				Type:    "https",
				Front:   "",
			}, {
				Address: "https://3.th.ooni.org/",
				Type:    "https",
				Front:   "",
			}}
			return output, true
		},

		MockDefaultHTTPClient: func() model.HTTPClient {
			return client
		},

		MockFetchPsiphonConfig: nil,

		MockFetchTorTargets: nil,

		MockKeyValueStore: nil,

		MockLookupASN: func(ip string) (asn uint, org string, err error) {
			// We're using IP addresses inside the 130.192.91.x address space and we should
			// make each of them a different ASN. Everything else is not mapped.
			if !strings.HasPrefix(ip, "130.192.91.") {
				return 0, "", errors.New("geoipx: no such ASN")
			}
			asString := strings.TrimPrefix(ip, "130.192.91.")
			asNum := runtimex.Try1(strconv.Atoi(asString))
			return uint(asNum), "Org " + asString, nil
		},

		MockLogger: func() model.Logger {
			return logger
		},

		MockMaybeResolverIP: nil,

		MockProbeASNString: nil,

		MockProbeCC: nil,

		MockProbeIP: nil,

		MockProbeNetworkName: nil,

		MockProxyURL: nil,

		MockResolverIP: func() string {
			return netemx.QAEnvDefaultISPResolverAddress
		},

		MockSoftwareName: nil,

		MockSoftwareVersion: nil,

		MockTempDir: nil,

		MockTorArgs: nil,

		MockTorBinary: nil,

		MockTunnelDir: nil,

		MockUserAgent: func() string {
			return model.HTTPHeaderUserAgent
		},

		MockNewExperimentBuilder: nil,

		MockNewSubmitter: nil,

		MockCheckIn: nil,
	}
}
