package model

import "time"

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

// OOAPICheckInInfoNDT contains the NDT
// part of OOAPICheckInInfo.
type OOAPICheckInInfoNDT struct {
	// ReportID is the report ID the probe should use.
	ReportID string `json:"report_id"`
}

// OOAPICheckInInfoDash contains the Dash
// part of OOAPICheckInInfo.
type OOAPICheckInInfoDash struct {
	// ReportID is the report ID the probe should use.
	ReportID string `json:"report_id"`
}

// OOAPICheckInInfoHHFM contains the HHFM
// part of OOAPICheckInInfo.
type OOAPICheckInInfoHHFM struct {
	// Report ID is the report ID the probe should use.
	ReportID string `json:"report_id"`
}

// OOAPICheckInInfoHIRL contains the HIRL
// part of OOAPICheckInInfo.
type OOAPICheckInInfoHIRL struct {
	// Report ID is the report ID the probe should use.
	ReportID string `json:"report_id"`
}

// OOAPICheckInInfoFacebookMessenger contains the FBMessenger
// part of OOAPICheckInInfo.
type OOAPICheckInInfoFacebookMessenger struct {
	// ReportID is the report ID the probe should use.
	ReportID string `json:"report_id"`
}

// OOAPICheckInInfoSignal contains the Signal
// part of OOAPICheckInInfo.
type OOAPICheckInInfoSignal struct {
	// ReportID is the report ID the probe should use.
	ReportID string `json:"report_id"`
}

// OOAPICheckInInfoTelegram contains the Telegram
// part of OOAPICheckInInfo.
type OOAPICheckInInfoTelegram struct {
	// ReportID is the report ID the probe should use.
	ReportID string `json:"report_id"`
}

// OOAPICheckInInfoWhatsapp contains the Whatsapp
// part of OOAPICheckInInfo.
type OOAPICheckInInfoWhatsapp struct {
	// ReportID is the report ID the probe should use.
	ReportID string `json:"report_id"`
}

// OOAPICheckInInfoPsiphon contains the Psiphon
// part of OOAPICheckInInfo.
type OOAPICheckInInfoPsiphon struct {
	// ReportID is the report ID the probe should use.
	ReportID string `json:"report_id"`
}

// OOAPICheckInInfoTor contains the Tor
// part of OOAPICheckInInfo.
type OOAPICheckInInfoTor struct {
	// ReportID is the report ID the probe should use.
	ReportID string `json:"report_id"`
}

// OOAPICheckInInfoDNSCheck contains the DNSCheck
// part of OOAPICheckInInfo.
type OOAPICheckInInfoDNSCheck struct {
	// ReportID is the report ID the probe should use.
	ReportID string `json:"report_id"`
}

// OOAPICheckInInfoStunReachability contains the StunReachability
// part of OOAPICheckInInfo.
type OOAPICheckInInfoStunReachability struct {
	// ReportID is the report ID the probe should use.
	ReportID string `json:"report_id"`
}

// OOAPICheckInInfoTorsf contains the Torsf
// part of OOAPICheckInInfo.
type OOAPICheckInInfoTorsf struct {
	// ReportID is the report ID the probe should use.
	ReportID string `json:"report_id"`
}

// OOAPICheckInInfoVanillaTor contains the VanillaTor
// part of OOAPICheckInInfo.
type OOAPICheckInInfoVanillaTor struct {
	// ReportID is the report ID the probe should use.
	ReportID string `json:"report_id"`
}

// OOAPICheckInInfoRiseupVPN contains the RiseupVPN
// part of OOAPICheckInInfo.
type OOAPICheckInInfoRiseupVPN struct {
	// ReportID is the report ID the probe should use.
	ReportID string `json:"report_id"`
}

// OOAPICheckInNettests contains nettest information returned by the checkin API call.
type OOAPICheckInNettests struct {
	// WebConnectivity contains WebConnectivity related information.
	WebConnectivity *OOAPICheckInInfoWebConnectivity `json:"web_connectivity"`

	// Dash contains Dash related information.
	Dash *OOAPICheckInInfoDash `json:"dash"`

	// NDT contains NDT related information.
	NDT *OOAPICheckInInfoNDT `json:"ndt"`

	// HHFM contains the HHFM related information.
	HHFM *OOAPICheckInInfoHHFM `json:"http_header_field_manipulation"`

	// HIRL contains the HIRL related information.
	HIRL *OOAPICheckInInfoHIRL `json:"http_invalid_request_line"`

	// FacebookMessenger contaings Facebook Messenger related information.
	FacebookMessenger *OOAPICheckInInfoFacebookMessenger `json:"facebook_messenger"`

	// Signal contains Signal related information.
	// TODO: Add Signal to the check-in API response
	Signal *OOAPICheckInInfoSignal `json:"signal"`

	// Telegram contains Telegram related information.
	Telegram *OOAPICheckInInfoTelegram `json:"telegram"`

	// Whatsapp contains Whatsapp related information.
	Whatsapp *OOAPICheckInInfoWhatsapp `json:"whatsapp"`

	// Psiphon contains Psiphon related information.
	Psiphon *OOAPICheckInInfoPsiphon `json:"psiphon"`

	// Tor contains Tor related information.
	Tor *OOAPICheckInInfoTor `json:"tor"`

	// DNSCheck contains DNSCheck related information.
	DNSChck *OOAPICheckInInfoDNSCheck `json:"dnscheck"`

	// StunReachability contains StunReachability related information.
	StunReachability *OOAPICheckInInfoStunReachability `json:"stun_reachability"`

	// Torsf contains Torsf related information.
	Torsf *OOAPICheckInInfoTorsf `json:"torsf"`

	// VanillaTor contains VanillaTor related information.
	VanillaTor *OOAPICheckInInfoVanillaTor `json:"vanilla_tor"`

	// RiseupVPN contains RiseupVPN related information.
	RiseupVPN *OOAPICheckInInfoRiseupVPN `json:"riseupvpn"`
}

// OOAPICheckInResult is the result returned by the checkin API.
type OOAPICheckInResult struct {
	// ProbeASN contains the probe's ASN.
	ProbeASN string `json:"probe_asn"`

	// ProbeCC contains the probe's CC.
	ProbeCC string `json:"probe_cc"`

	// Tests contains information about nettests.
	Tests OOAPICheckInNettests `json:"tests"`

	// V is the version.
	V int64 `json:"v"`
}

// OOAPICheckReportIDResponse is the check-report-id API response.
type OOAPICheckReportIDResponse struct {
	Error string `json:"error"`
	Found bool   `json:"found"`
	V     int64  `json:"v"`
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

const (
	// OOAPIReportDefaultDataFormatVersion is the default data format version.
	//
	// See https://github.com/ooni/spec/tree/master/data-formats#history.
	OOAPIReportDefaultDataFormatVersion = "0.2.0"

	// DefaultFormat is the default format
	OOAPIReportDefaultFormat = "json"
)

// OOAPIReportTemplate is the template for opening a report
type OOAPIReportTemplate struct {
	// DataFormatVersion is unconditionally set to DefaultDataFormatVersion
	// and you don't need to be concerned about it.
	DataFormatVersion string `json:"data_format_version"`

	// Format is unconditionally set to `json` and you don't need
	// to be concerned about it.
	Format string `json:"format"`

	// ProbeASN is the probe's autonomous system number (e.g. `AS1234`)
	ProbeASN string `json:"probe_asn"`

	// ProbeCC is the probe's country code (e.g. `IT`)
	ProbeCC string `json:"probe_cc"`

	// SoftwareName is the app name (e.g. `measurement-kit`)
	SoftwareName string `json:"software_name"`

	// SoftwareVersion is the app version (e.g. `0.9.1`)
	SoftwareVersion string `json:"software_version"`

	// TestName is the test name (e.g. `ndt`)
	TestName string `json:"test_name"`

	// TestStartTime contains the test start time
	TestStartTime string `json:"test_start_time"`

	// TestVersion is the test version (e.g. `1.0.1`)
	TestVersion string `json:"test_version"`
}

// OOAPICollectorOpenResponse is the response returned by the open report API.
type OOAPICollectorOpenResponse struct {
	// BackendVersion is the backend version.
	BackendVersion string `json:"backend_version"`

	// ReportID is the report ID.
	ReportID string `json:"report_id"`

	// SupportedFormats contains supported formats.
	SupportedFormats []string `json:"supported_formats"`
}

// OOAPICollectorUpdateRequest is a request for the collector update API.
type OOAPICollectorUpdateRequest struct {
	// Format is the Content's data format
	Format string `json:"format"`

	// Content is the actual report
	Content any `json:"content"`
}

// OOAPICollectorUpdateResponse is the response from the collector update API.
type OOAPICollectorUpdateResponse struct {
	// MeasurementID is the measurement ID.
	MeasurementID string `json:"measurement_id"`
}

// OOAPILoginCredentials contains the login credentials
type OOAPILoginCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// OOAPILoginAuth contains authentication info
type OOAPILoginAuth struct {
	Expire time.Time `json:"expire"`
	Token  string    `json:"token"`
}

// OOAPIMeasurementMetaConfig contains configuration for GetMeasurementMeta.
type OOAPIMeasurementMetaConfig struct {
	// ReportID is the mandatory report ID.
	ReportID string

	// Full indicates whether we also want the full measurement body.
	Full bool

	// Input is the optional input.
	Input string
}

// OOAPIMeasurementMeta contains measurement metadata.
type OOAPIMeasurementMeta struct {
	// Fields returned by the API server whenever we are
	// calling /api/v1/measurement_meta.
	Anomaly              bool      `json:"anomaly"`
	CategoryCode         string    `json:"category_code"`
	Confirmed            bool      `json:"confirmed"`
	Failure              bool      `json:"failure"`
	Input                *string   `json:"input"`
	MeasurementStartTime time.Time `json:"measurement_start_time"`
	ProbeASN             int64     `json:"probe_asn"`
	ProbeCC              string    `json:"probe_cc"`
	ReportID             string    `json:"report_id"`
	Scores               string    `json:"scores"`
	TestName             string    `json:"test_name"`
	TestStartTime        time.Time `json:"test_start_time"`

	// This field is only included if the user has specified
	// the config.Full option, otherwise it's empty.
	RawMeasurement string `json:"raw_measurement"`
}

// OOAPIProbeMetadata contains metadata about a probe. This message is
// included into a bunch of messages sent to orchestra.
type OOAPIProbeMetadata struct {
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
func (m OOAPIProbeMetadata) Valid() bool {
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

// OOAPIRegisterRequest is a request to the register API.
type OOAPIRegisterRequest struct {
	OOAPIProbeMetadata
	Password string `json:"password"`
}

// OOAPIRegisterResponse is a reponse from the register API.
type OOAPIRegisterResponse struct {
	ClientID string `json:"client_id"`
}

// OOAPIURLListResult is the result of the /api/v1/test-list/urls API call.
type OOAPIURLListResult struct {
	Metadata OOAPIURLListMetadata `json:"metadata"`
	Results  []OOAPIURLInfo       `json:"results"`
}

// OONIAPIURLListMetadata contains metadata included
// inside the OOAPIURLListResult struct.
type OOAPIURLListMetadata struct {
	Count int64 `json:"count"`
}
