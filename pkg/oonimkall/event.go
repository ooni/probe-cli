package oonimkall

type eventEmpty struct{}

// eventFailure contains information on a failure.
type eventFailure struct {
	Failure string `json:"failure"`
}

// eventLog is an event containing a log message.
type eventLog struct {
	LogLevel string `json:"log_level"`
	Message  string `json:"message"`
}

type eventMeasurementGeneric struct {
	Failure string `json:"failure,omitempty"`
	Idx     int64  `json:"idx"`
	Input   string `json:"input"`
	JSONStr string `json:"json_str,omitempty"`
}

type eventStatusEnd struct {
	DownloadedKB float64 `json:"downloaded_kb"`
	Failure      string  `json:"failure"`
	UploadedKB   float64 `json:"uploaded_kb"`
}

type eventStatusGeoIPLookup struct {
	ProbeASN         string `json:"probe_asn"`
	ProbeCC          string `json:"probe_cc"`
	ProbeIP          string `json:"probe_ip"`
	ProbeNetworkName string `json:"probe_network_name"`
}

// eventStatusProgress reports progress information.
type eventStatusProgress struct {
	Message    string  `json:"message"`
	Percentage float64 `json:"percentage"`
}

type eventStatusReportGeneric struct {
	ReportID string `json:"report_id"`
}

type eventStatusResolverLookup struct {
	ResolverASN         string `json:"resolver_asn"`
	ResolverIP          string `json:"resolver_ip"`
	ResolverNetworkName string `json:"resolver_network_name"`
}

// event is an event emitted by a task. This structure extends the event
// described by MK v0.10.9 FFI API (https://git.io/Jv4Rv).
type event struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}
