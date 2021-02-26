package apimodel

// CheckInRequestWebConnectivity contains WebConnectivity
// specific parameters to include into CheckInRequest
type CheckInRequestWebConnectivity struct {
	CategoryCodes []string `json:"category_codes"`
}

// CheckInRequest is the check-in API request
type CheckInRequest struct {
	Charging        bool                          `json:"charging"`
	OnWiFi          bool                          `json:"on_wifi"`
	Platform        string                        `json:"platform"`
	ProbeASN        string                        `json:"probe_asn"`
	ProbeCC         string                        `json:"probe_cc"`
	RunType         string                        `json:"run_type"`
	SoftwareName    string                        `json:"software_name"`
	SoftwareVersion string                        `json:"software_version"`
	WebConnectivity CheckInRequestWebConnectivity `json:"web_connectivity"`
}

// CheckInResponseURLInfo contains information about an URL.
type CheckInResponseURLInfo struct {
	CategoryCode string `json:"category_code"`
	CountryCode  string `json:"country_code"`
	URL          string `json:"url"`
}

// CheckInResponseWebConnectivity contains WebConnectivity
// specific information of a CheckInResponse
type CheckInResponseWebConnectivity struct {
	ReportID string                   `json:"report_id"`
	URLs     []CheckInResponseURLInfo `json:"urls"`
}

// CheckInResponse is the check-in API response
type CheckInResponse struct {
	ProbeASN string               `json:"probe_asn"`
	ProbeCC  string               `json:"probe_cc"`
	Tests    CheckInResponseTests `json:"tests"`
	V        int64                `json:"v"`
}

// CheckInResponseTests contains configuration for tests
type CheckInResponseTests struct {
	WebConnectivity CheckInResponseWebConnectivity `json:"web_connectivity"`
}
