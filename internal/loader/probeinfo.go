package loader

import "github.com/ooni/probe-cli/v3/internal/model"

// ProbeInfo contains information about the probe making the request.
type ProbeInfo struct {
	// Charging indicate whether the probe is charging.
	Charging bool `json:"charging"`

	// OnWiFi indicate if the probe is connected to a WiFi.
	OnWiFi bool `json:"on_wifi"`

	// Platform of the probe.
	Platform string `json:"platform"`

	// ProbeASN is the probe ASN.
	ProbeASN string `json:"probe_asn"`

	// ProbeCC is the probe country code.
	ProbeCC string `json:"probe_cc"`

	// RunType indicated whether the run is "timed" or "manual".
	RunType model.RunType `json:"run_type"`

	// SoftwareName of the probe.
	SoftwareName string `json:"software_name"`

	// SoftwareVersion of the probe.
	SoftwareVersion string `json:"software_version"`
}
