package nwcth

import (
	"context"
	"errors"
	"net/url"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/websteps"
	"github.com/ooni/probe-cli/v3/internal/engine/netx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

type (
	CtrlRequest     = websteps.CtrlRequest
	ControlResponse = websteps.ControlResponse
)

var ErrInternalServer = errors.New("Internal server failure")

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
func Measure(ctx context.Context, creq *CtrlRequest, config *Config) (*ControlResponse, error) {
	var (
		URL *url.URL
		err error
	)
	resolver := config.resolver
	if resolver == nil {
		// use a central resolver
		resolver = newResolver()
	}
	checker := config.checker
	if checker == nil {
		checker = &DefaultInitChecker{resolver: resolver}
	}
	URL, err = checker.InitialChecks(creq.HTTPRequest)
	if err != nil {
		// return a valid response in case of NXDOMAIN so the probe can compare the failure
		if err == ErrNoSuchHost {
			return newDNSFailedResponse(err, creq.HTTPRequest), nil
		}
		return nil, err
	}
	explorer := config.explorer
	if explorer == nil {
		explorer = &DefaultExplorer{resolver: resolver}
	}
	rts, err := explorer.Explore(URL, creq.HTTPRequestHeaders)
	if err != nil {
		return nil, ErrInternalServer
	}
	generator := config.generator
	if generator == nil {
		generator = &DefaultGenerator{resolver: resolver}
	}
	meas, err := generator.Generate(ctx, rts, creq.Addrs)
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
