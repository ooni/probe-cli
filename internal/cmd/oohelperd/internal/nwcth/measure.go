package nwcth

import (
	"context"
	"errors"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/nwebconnectivity"
)

// ControlResponse is the response from the control service.
type ControlResponse struct {
	URLMeasurements []*URLMeasurement `json:"urls"`
}

type (
	// ControlRequest is the request sent to the test helper
	ControlRequest = nwebconnectivity.ControlRequest

	// HTTPMeasurement contains the measurement for one URL, and one IP endpoint,
	// using HTTP over TCP.
	HTTPMeasurement = nwebconnectivity.ControlHTTPMeasurement

	// H3Measurement contains the measurement for one URL, and one IP endpoint,
	// using HTTP over QUIC (HTTP/3).
	H3Measurement = nwebconnectivity.ControlH3Measurement
)

// ErrNoValidIP means that the DNS step failed and the client did not provide IP endpoints for testing.
var ErrNoValidIP = errors.New("no valid IP address to measure")

// supportedQUICVersion are the H3 over QUIC versions we currently support
var supportedQUICVersions = map[string]bool{
	"h3":    true,
	"h3-29": true,
}

func Measure(ctx context.Context, creq *ControlRequest) (*ControlResponse, error) {
	var (
		URL *url.URL
		err error
	)
	URL, err = InitialChecks(creq.HTTPRequest)
	if err != nil {
		return nil, err
	}
	rts, err := Explore(URL)
	if err != nil {
		return nil, err
	}
	meas, err := Generate(ctx, rts)
	if err != nil {
		return nil, err
	}
	return &ControlResponse{URLMeasurements: meas}, nil
}
