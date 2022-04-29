package model

//
// Data structures used to speak with the OONI API.
//

// OOAPICheckInConfigWebConnectivity is the configuration for the WebConnectivity test
type OOAPICheckInConfigWebConnectivity struct {
	CategoryCodes []string `json:"category_codes"` // CategoryCodes is an array of category codes
}

// OOAPICheckInConfig contains configuration for calling the checkin API.
type OOAPICheckInConfig struct {
	Charging        bool                              `json:"charging"`         // Charging indicate if the phone is actually charging
	OnWiFi          bool                              `json:"on_wifi"`          // OnWiFi indicate if the phone is actually connected to a WiFi network
	Platform        string                            `json:"platform"`         // Platform of the probe
	ProbeASN        string                            `json:"probe_asn"`        // ProbeASN is the probe country code
	ProbeCC         string                            `json:"probe_cc"`         // ProbeCC is the probe country code
	RunType         RunType                           `json:"run_type"`         // RunType
	SoftwareName    string                            `json:"software_name"`    // SoftwareName of the probe
	SoftwareVersion string                            `json:"software_version"` // SoftwareVersion of the probe
	WebConnectivity OOAPICheckInConfigWebConnectivity `json:"web_connectivity"` // WebConnectivity class contain an array of categories
}

// OOAPICheckInInfoWebConnectivity contains the array of URLs returned by the checkin API
type OOAPICheckInInfoWebConnectivity struct {
	ReportID string         `json:"report_id"`
	URLs     []OOAPIURLInfo `json:"urls"`
}

// OOAPICheckInInfo contains the return test objects from the checkin API
type OOAPICheckInInfo struct {
	WebConnectivity *OOAPICheckInInfoWebConnectivity `json:"web_connectivity"`
}

// OOAPIService describes a backend service.
//
// The fields of this struct have the meaning described in v2.0.0 of the OONI
// bouncer specification defined by
// https://github.com/ooni/spec/blob/master/backends/bk-004-bouncer.md.
type OOAPIService struct {
	// Address is the address of the server.
	Address string `json:"address"`

	// Type is the type of the service.
	Type string `json:"type"`

	// Front is the front to use with "cloudfront" type entries.
	Front string `json:"front,omitempty"`
}

// OOAPITorTarget is a target for the tor experiment.
type OOAPITorTarget struct {
	// Address is the address of the target.
	Address string `json:"address"`

	// Name is the name of the target.
	Name string `json:"name"`

	// Params contains optional params for, e.g., pluggable transports.
	Params map[string][]string `json:"params"`

	// Protocol is the protocol to use with the target.
	Protocol string `json:"protocol"`

	// Source is the source from which we fetched this specific
	// target. Whenever the source is non-empty, we will treat
	// this specific target as a private target.
	Source string `json:"source"`
}

// OOAPIURLInfo contains info on a test lists URL
type OOAPIURLInfo struct {
	CategoryCode string `json:"category_code"`
	CountryCode  string `json:"country_code"`
	URL          string `json:"url"`
}

// OOAPIURLListConfig contains configuration for fetching the URL list.
type OOAPIURLListConfig struct {
	Categories  []string // Categories to query for (empty means all)
	CountryCode string   // CountryCode is the optional country code
	Limit       int64    // Max number of URLs (<= 0 means no limit)
}
