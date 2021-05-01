package apimodel

// CheckReportIDRequest is the CheckReportID request.
type CheckReportIDRequest struct {
	ReportID string `query:"report_id" required:"true"`
}

// CheckReportIDResponse is the CheckReportID response.
type CheckReportIDResponse struct {
	Error string `json:"error"`
	Found bool   `json:"found"`
	V     int64  `json:"v"`
}
