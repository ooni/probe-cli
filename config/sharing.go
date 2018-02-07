package config

// Sharing settings
type Sharing struct {
	IncludeIP        bool `json:"include_ip"`
	IncludeASN       bool `json:"include_asn"`
	IncludeGPS       bool `json:"include_gps"`
	UploadResults    bool `json:"upload_results"`
	SendCrashReports bool `json:"send_crash_reports"`
}
