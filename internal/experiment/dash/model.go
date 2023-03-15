package dash

//
// Constants and data model.
//
// See the spec: https://github.com/ooni/spec/blob/master/nettests/ts-021-dash.md.
//

import (
	"errors"
	"time"
)

const (
	// defaultTimeout is the default timeout for the whole experiment.
	defaultTimeout = 120 * time.Second

	// magicVersion encodes the version number of the tool we are
	// using according to the format originally used by Neubot. We
	// used "0.007000000" for Measurement Kit, which mapped to Neubot
	// v0.7.0. OONI pretends to be Neubot v0.8.0.
	magicVersion = "0.008000000"

	// testName is the name of the experiment.
	testName = "dash"

	// testVersion is the version of the experiment.
	testVersion = "0.14.0"

	// totalStep is the total number of steps we should run
	// during the download experiment.
	totalStep = 15
)

var (
	// errServerBusy is the error returned when the DASH server is busy.
	errServerBusy = errors.New("dash: server busy; try again later")

	// errHTTPRequest failed is the error returned when an HTTP request fails.
	errHTTPRequestFailed = errors.New("dash: request failed")
)

const (
	// negotiatePath is the URL path used to negotiate
	negotiatePath = "/negotiate/dash"

	// downloadPath is the URL path used to request DASH segments. You can
	// append to this path an integer indicating how many bytes you would like
	// the server to send you as part of the next chunk.
	downloadPath = "/dash/download/"

	// collectPath is the URL path used to collect
	collectPath = "/collect/dash"
)

// defaultRates contains the default DASH rates in kbit/s.
var defaultRates = []int64{
	100, 150, 200, 250, 300, 400, 500, 700, 900, 1200, 1500, 2000,
	2500, 3000, 4000, 5000, 6000, 7000, 10000, 20000,
}

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
