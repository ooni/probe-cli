package model

// CheckInConfigWebConnectivity is the configuration for the WebConnectivity test
type CheckInConfigWebConnectivity struct {
	CategoryCodes []string `json:"category_codes"` // CategoryCodes is an array of category codes
}

// CheckInConfig contains configuration for calling the checkin API.
type CheckInConfig struct {
	Charging        bool                         `json:"charging"`         // Charging indicate if the phone is actually charging
	OnWiFi          bool                         `json:"on_wifi"`          // OnWiFi indicate if the phone is actually connected to a WiFi network
	Platform        string                       `json:"platform"`         // Platform of the probe
	ProbeASN        string                       `json:"probe_asn"`        // ProbeASN is the probe country code
	ProbeCC         string                       `json:"probe_cc"`         // ProbeCC is the probe country code
	RunType         string                       `json:"run_type"`         // RunType
	SoftwareName    string                       `json:"software_name"`    // SoftwareName of the probe
	SoftwareVersion string                       `json:"software_version"` // SoftwareVersion of the probe
	WebConnectivity CheckInConfigWebConnectivity `json:"web_connectivity"` // WebConnectivity class contain an array of categories
}
