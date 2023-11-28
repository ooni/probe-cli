package webconnectivityqa

import (
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netemx"
)

// newSession creates a new [model.ExperimentSession].
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
			return netemx.ISPResolverAddress
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
