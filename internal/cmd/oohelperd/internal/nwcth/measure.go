package nwcth

import (
	"context"
	"errors"
	"net/url"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/netx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// ControlRequest is the request sent by the probe
type ControlRequest struct {
	HTTPRequest        string              `json:"http_request"`
	HTTPRequestHeaders map[string][]string `json:"http_request_headers"`
	Addrs              []string            `json:"addrs"`
}

// ControlResponse is the response from the control service.
type ControlResponse struct {
	URLMeasurements []*URLMeasurement `json:"urls"`
}

var ErrInternalServer = errors.New("Internal server failure")

// supportedQUICVersions are the H3 over QUIC versions we currently support
var supportedQUICVersions = map[string]bool{
	"h3":    true,
	"h3-29": true,
}

// Config contains the building blocks of the testhelper algorithm
type Config struct {
	checker   InitChecker
	explorer  Explorer
	generator Generator
	resolver  netxlite.Resolver
}

// Measure performs the three consecutive steps of the testhelper algorithm:
// 1. InitialChecks
// 2. Explore
// 3. Generate
func Measure(ctx context.Context, creq *ControlRequest, config *Config) (*ControlResponse, error) {
	var (
		URL *url.URL
		err error
	)
	if config.resolver == nil {
		// use a central resolver
		config.resolver = newResolver()
	}
	if config.checker == nil {
		config.checker = &DefaultInitChecker{resolver: config.resolver}
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
		config.explorer = &DefaultExplorer{resolver: config.resolver}
	}
	rts, err := config.explorer.Explore(URL, creq.HTTPRequestHeaders)
	if err != nil {
		return nil, ErrInternalServer
	}
	if config.generator == nil {
		config.generator = &DefaultGenerator{resolver: config.resolver}
	}
	meas, err := config.generator.Generate(ctx, rts, creq.Addrs)
	if err != nil {
		return nil, err
	}
	return &ControlResponse{URLMeasurements: meas}, nil
}

// newDNSFailedResponse creates a new response with one URLMeasurement entry
// indicating that the DNS step failed
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

// newResolver creates a new DNS resolver instance
func newResolver() netxlite.Resolver {
	childResolver, err := netx.NewDNSClient(netx.Config{Logger: log.Log}, "doh://google")
	runtimex.PanicOnError(err, "NewDNSClient failed")
	var r netxlite.Resolver = childResolver
	r = &netxlite.IDNAResolver{Resolver: r}
	return r
}
