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
	var (
		URL *url.URL
		err error
	)
	if config.checker == nil {
		config.checker = &defaultInitChecker{}
	}
	URL, err = config.checker.InitialChecks(creq.HTTPRequest)
	if err != nil {
		return nil, err
	}
	if config.explorer == nil {
		config.explorer = &defaultExplorer{}
	}
	rts, err := config.explorer.Explore(URL)
	if err != nil {
		return nil, err
	}
	if config.generator == nil {
		config.generator = &defaultGenerator{}
	}
	meas, err := config.generator.Generate(ctx, rts)
	if err != nil {
		return nil, err
	}
	return &ControlResponse{URLMeasurements: meas}, nil
}
