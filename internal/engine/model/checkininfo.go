package model

// OOAPICheckInInfoWebConnectivity contains the array of URLs returned by the checkin API
type OOAPICheckInInfoWebConnectivity struct {
	ReportID string         `json:"report_id"`
	URLs     []OOAPIURLInfo `json:"urls"`
}

// OOAPICheckInInfo contains the return test objects from the checkin API
type OOAPICheckInInfo struct {
	WebConnectivity *OOAPICheckInInfoWebConnectivity `json:"web_connectivity"`
}
