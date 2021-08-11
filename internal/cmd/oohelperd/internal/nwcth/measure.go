package nwcth

import (
	"context"
	"errors"
	"fmt"
	"log"
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
	if URL, err = InitialChecks(creq.HTTPRequest); err != nil {
		log.Fatalf("initial checks failed: %s", err.Error())
	}
	rts, err := Explore(URL)
	if err != nil {
		log.Fatalf("explore failed: %s", err.Error())
	}
	meas, err := Generate(ctx, rts)
	if err != nil {
		log.Fatalf("generate failed: %s", err.Error())
	}
	for _, m := range meas {
		fmt.Printf("# %s\n", m.URL)
		fmt.Printf("method: %s\n", m.RoundTrip.Request.Method)
		fmt.Printf("url: %s\n", m.RoundTrip.Request.URL.String())
		fmt.Printf("headers: %+v\n", m.RoundTrip.Request.Header)
		fmt.Printf("dns: %+v\n", m.DNS)
	}
	return &ControlResponse{URLMeasurements: meas}, nil
}
