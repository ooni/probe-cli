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

var ErrInternalServer = errors.New("Internal server failure")

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
		// return a valid response in case of NXDOMAIN so the probe can compare the failure
		if err == ErrNoSuchHost {
			return newDNSFailedResponse(err, creq.HTTPRequest), nil
		}
		return nil, err
	}
	if config.explorer == nil {
		config.explorer = &defaultExplorer{}
	}
	rts, err := config.explorer.Explore(URL)
	if err != nil {
		return nil, ErrInternalServer
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

func newDNSFailedResponse(err error, URL string) *ControlResponse {
	resp := &ControlResponse{}
	m := &URLMeasurement{
		URL: URL,
		DNS: &DNSMeasurement{
			Failure: newfailure(err),
		},
	}
	resp.URLMeasurements = append(resp.URLMeasurements, m)
	return resp
}
