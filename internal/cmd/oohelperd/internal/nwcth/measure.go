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

// ControlRequest is the request sent to the test helper
type ControlRequest = nwebconnectivity.ControlRequest

// ErrNoValidIP means that the DNS step failed and the client did not provide IP endpoints for testing.
var ErrNoValidIP = errors.New("no valid IP address to measure")

// supportedQUICVersion are the H3 over QUIC versions we currently support
var supportedQUICVersions = map[string]bool{
	"h3":    true,
	"h3-29": true,
}

type Config struct {
	checker   InitChecker
	explorer  Explorer
	generator Generator
}

func Measure(ctx context.Context, creq *ControlRequest, config *Config) (*ControlResponse, error) {
	resp := &ControlResponse{}
	var (
		URL *url.URL
		err error
	)
	if config.checker == nil {
		config.checker = &defaultInitChecker{}
	}
	URL, err = config.checker.InitialChecks(creq.HTTPRequest)
	// return a valid response even in the error case so the probe can compare the failure
	m := &URLMeasurement{
		URL: creq.HTTPRequest,
		DNS: &DNSMeasurement{
			Failure: newfailure(err),
		},
	}
	resp.URLMeasurements = append(resp.URLMeasurements, m)
	if err != nil {
		return resp, err
	}
	if config.explorer == nil {
		config.explorer = &defaultExplorer{}
	}
	rts, err := config.explorer.Explore(URL)
	if err != nil {
		// TODO(kelmenhorst,bassosimone): what happens here?
		return resp, err
	}
	if config.generator == nil {
		config.generator = &defaultGenerator{}
	}
	meas, err := config.generator.Generate(ctx, rts)
	if err != nil {
		return nil, err
	}
	// TODO(kelmenhorst,bassosimone): Is it ok to replace the URLMeasurement from InitialChecks here?
	return &ControlResponse{URLMeasurements: meas}, nil
}
