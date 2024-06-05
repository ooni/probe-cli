package wireguard

// TestKeys contains the experiment's result.
//
// This is what will end up into the Measurement.TestKeys field
// when you run this experiment.
//
// In other words, the variables in this struct will be
// the specific results of this experiment.
type TestKeys struct {
	Success       bool            `json:"success"`
	Failure       *string         `json:"failure"`
	NetworkEvents []*Event        `json:"network_events"`
	URLGet        []*URLGetResult `json:"urlget"`
}

// URLGetResult is the result of fetching a URL via the wireguard tunnel,
// using the standard library.
type URLGetResult struct {
	ByteCount  int     `json:"bytes,omitempty"`
	Error      string  `json:"error,omitempty"`
	Failure    *string `json:"failure"`
	StatusCode int     `json:"status_code"`
	T0         float64 `json:"t0"`
	T          float64 `json:"t"`
	URL        string  `json:"url"`
}
