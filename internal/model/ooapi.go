package model

//
// OONI API data model.
//
// See https://api.ooni.io/apidocs/.
//

// OOAPICheckInConfigWebConnectivity is the WebConnectivity
// portion of OOAPICheckInConfig.
type OOAPICheckInConfigWebConnectivity struct {
	// CategoryCodes contains an array of category codes
	CategoryCodes []string `json:"category_codes"`
}

// OOAPICheckInConfig contains config for a checkin API call.
type OOAPICheckInConfig struct {
	// Charging indicate whether the phone is charging.
	Charging bool `json:"charging"`

	// OnWiFi indicate if the phone is connected to a WiFi.
	OnWiFi bool `json:"on_wifi"`

	// Platform of the probe.
	Platform string `json:"platform"`

	// ProbeASN is the probe ASN.
	ProbeASN string `json:"probe_asn"`

	// ProbeCC is the probe country code.
	ProbeCC string `json:"probe_cc"`

	// RunType indicated whether the run is "timed" or "manual".
	RunType RunType `json:"run_type"`

	// SoftwareName of the probe.
	SoftwareName string `json:"software_name"`

	// SoftwareVersion of the probe.
	SoftwareVersion string `json:"software_version"`

	// WebConnectivity contains WebConnectivity information.
	WebConnectivity OOAPICheckInConfigWebConnectivity `json:"web_connectivity"`
}

// OOAPICheckInInfoWebConnectivity contains the WebConnectivity
// part of OOAPICheckInInfo.
type OOAPICheckInInfoWebConnectivity struct {
	// ReportID is the report ID the probe should use.
	ReportID string `json:"report_id"`

	// URLs contains the URL to measure.
	URLs []OOAPIURLInfo `json:"urls"`
}

// OOAPICheckInInfo contains the information returned by the checkin API call.
type OOAPICheckInInfo struct {
	// WebConnectivity contains WebConnectivity related information.
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

// OOAPIURLInfo contains information on a test lists URL.
type OOAPIURLInfo struct {
	// CategoryCode is the URL's category (e.g., FEXP, POLT, HUMR).
	CategoryCode string `json:"category_code"`

	// CountryCode is the URL's ISO country code or ZZ for global URLs.
	CountryCode string `json:"country_code"`

	// URL is the string-serialized URL.
	URL string `json:"url"`
}

// OOAPIURLListConfig contains configuration for fetching the URL list.
type OOAPIURLListConfig struct {
	// Categories to query for (empty means all)
	Categories []string

	// CountryCode is the optional country code
	CountryCode string

	// Max number of URLs (<= 0 means no limit)
	Limit int64
}
