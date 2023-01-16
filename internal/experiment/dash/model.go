package dash

// clientResults contains the results measured by the client. This data
// structure is sent to the server in the collection phase.
//
// All the fields listed here are part of the original specification
// of DASH, except ServerURL, added in MK v0.10.6.
type clientResults struct {
	ConnectTime     float64 `json:"connect_time"`
	DeltaSysTime    float64 `json:"delta_sys_time"`
	DeltaUserTime   float64 `json:"delta_user_time"`
	Elapsed         float64 `json:"elapsed"`
	ElapsedTarget   int64   `json:"elapsed_target"`
	InternalAddress string  `json:"internal_address"`
	Iteration       int64   `json:"iteration"`
	Platform        string  `json:"platform"`
	Rate            int64   `json:"rate"`
	RealAddress     string  `json:"real_address"`
	Received        int64   `json:"received"`
	RemoteAddress   string  `json:"remote_address"`
	RequestTicks    float64 `json:"request_ticks"`
	ServerURL       string  `json:"server_url"`
	Timestamp       int64   `json:"timestamp"`
	UUID            string  `json:"uuid"`
	Version         string  `json:"version"`
}

// serverResults contains the server results. This data structure is sent
// to the client during the collection phase of DASH.
type serverResults struct {
	Iteration int64   `json:"iteration"`
	Ticks     float64 `json:"ticks"`
	Timestamp int64   `json:"timestamp"`
}

// negotiateRequest contains the request of negotiation
type negotiateRequest struct {
	DASHRates []int64 `json:"dash_rates"`
}

// negotiateResponse contains the response of negotiation
type negotiateResponse struct {
	Authorization string `json:"authorization"`
	QueuePos      int64  `json:"queue_pos"`
	RealAddress   string `json:"real_address"`
	Unchoked      int    `json:"unchoked"`
}
