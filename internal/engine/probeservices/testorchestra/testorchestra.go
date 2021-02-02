// Package testorchestra helps with testing the OONI orchestra API.
package testorchestra

import "github.com/ooni/probe-cli/v3/internal/engine/probeservices"

// MetadataFixture returns a valid metadata struct. This is mostly
// useful for testing. (We should see if we can make this private.)
func MetadataFixture() probeservices.Metadata {
	return probeservices.Metadata{
		Platform:        "linux",
		ProbeASN:        "AS15169",
		ProbeCC:         "US",
		SoftwareName:    "miniooni",
		SoftwareVersion: "0.1.0-dev",
		SupportedTests: []string{
			"web_connectivity",
		},
	}
}
