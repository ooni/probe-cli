package apimodel

// RegisterRequest is the request for the Register API.
type RegisterRequest struct {
	Metadata RegisterRequestMetadata `json:"metadata"`
	Password string                  `json:"password"`
}

// RegisterRequestMetadata contains metadata for the register request.
type RegisterRequestMetadata struct {
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

// RegisterResponse is the response from the Register API.
type RegisterResponse struct {
	ClientID string `json:"client_id"`
}
