package urlgetter

import (
	"context"
	"net"
	"net/url"
	"time"

	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// DNSLookup measures a dnslookup://domain/ URL.
func (rx *Runner) DNSLookup(ctx context.Context, config *Config, URL *url.URL) error {
	_, err := rx.DNSLookupOp(ctx, config, URL)
	return err
}

// DNSLookupResult contains the results of a DNS lookup.
type DNSLookupResult struct {
	// Address is the resolved address.
	Address string

	// Config is the original config.
	Config *Config

	// URL is the original URL.
	URL *url.URL
}

// endpoint returns an endpoint given the address and the URL scheme.
func (rx *DNSLookupResult) endpoint() (string, error) {
	// handle the case where there is an explicit port
	if port := rx.URL.Port(); port != "" {
		return net.JoinHostPort(rx.Address, port), nil
	}

	// use the scheme to deduce the port
	switch rx.URL.Scheme {
	case "http":
		return net.JoinHostPort(rx.Address, "80"), nil
	case "https":
		return net.JoinHostPort(rx.Address, "443"), nil
	case "dot":
		return net.JoinHostPort(rx.Address, "853"), nil
	default:
		return "", ErrUnknownURLScheme
	}
}

// DNSLookupOp resolves a domain name using the configured resolver.
func (rx *Runner) DNSLookupOp(ctx context.Context, config *Config, URL *url.URL) ([]*DNSLookupResult, error) {
	// TODO(bassosimone): choose the proper DNS resolver depending on configuration.
	return rx.DNSLookupGetaddrinfoOp(ctx, config, URL)
}

// DNSLookupGetaddrinfoOp performs a DNS lookup using getaddrinfo.
func (rx *Runner) DNSLookupGetaddrinfoOp(ctx context.Context, config *Config, URL *url.URL) ([]*DNSLookupResult, error) {
	// enforce timeout
	const timeout = 4 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// obtain the next trace index
	index := rx.IndexGen.Next()

	// create trace using the given underlying network
	trace := measurexlite.NewTrace(index, rx.Begin)
	trace.Netx = &netxlite.Netx{Underlying: rx.UNet}

	// obtain logger
	logger := rx.Session.Logger()

	// create resolver
	reso := trace.NewStdlibResolver(logger)

	// the domain to resolve is the URL's hostname
	domain := URL.Hostname()

	// start operation logger
	ol := logx.NewOperationLogger(logger, "[#%d] lookup %s using getaddrinfo", index, domain)

	// perform the lookup
	addrs, err := reso.LookupHost(ctx, domain)

	// append the DNS lookup results
	rx.TestKeys.AppendQueries(trace.DNSLookupsFromRoundTrip()...)

	// stop the operation logger
	ol.Stop(err)

	// manually set the failure and failed operation
	if err != nil {
		rx.TestKeys.MaybeSetFailedOperation(netxlite.DNSRoundTripOperation)
		rx.TestKeys.MaybeSetFailure(err.Error())
		return nil, err
	}

	// emit results
	var results []*DNSLookupResult
	for _, addr := range addrs {
		results = append(results, &DNSLookupResult{
			Address: addr,
			Config:  config,
			URL:     URL,
		})
	}
	return results, nil
}
