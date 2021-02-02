package probeservices

// Metadata contains metadata about a probe. This message is
// included into a bunch of messages sent to orchestra.
type Metadata struct {
	AvailableBandwidth string   `json:"available_bandwidth,omitempty"`
	DeviceToken        string   `json:"device_token,omitempty"`
	Language           string   `json:"language,omitempty"`
	NetworkType        string   `json:"network_type,omitempty"`
	Platform           string   `json:"platform"`
	ProbeASN           string   `json:"probe_asn"`
	ProbeCC            string   `json:"probe_cc"`
	ProbeFamily        string   `json:"probe_family,omitempty"`
	ProbeTimezone      string   `json:"probe_timezone,omitempty"`
	SoftwareName       string   `json:"software_name"`
	SoftwareVersion    string   `json:"software_version"`
	SupportedTests     []string `json:"supported_tests"`
}

// Valid returns true if metadata is valid, false otherwise. Metadata is
// considered valid if all the mandatory fields are not empty. If a field
// is marked `json:",omitempty"` in the structure definition, then it's
// for sure mandatory. The "device_token" field is mandatory only if the
// platform is "ios" or "android", because there's currently no device
// token that we know of for desktop devices.
func (m Metadata) Valid() bool {
	if m.ProbeCC == "" {
		return false
	}
	if m.ProbeASN == "" {
		return false
	}
	if m.Platform == "" {
		return false
	}
	if m.SoftwareName == "" {
		return false
	}
	if m.SoftwareVersion == "" {
		return false
	}
	if len(m.SupportedTests) < 1 {
		return false
	}
	switch m.Platform {
	case "ios", "android":
		if m.DeviceToken == "" {
			return false
		}
	}
	return true
}
