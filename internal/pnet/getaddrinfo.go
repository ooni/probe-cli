package pnet

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// Getaddrinfo returns a [Stage] that resolves domain names using getaddrinfo(3).
func Getaddrinfo() Stage[DNSQuery, Endpoint] {
	return stageForAction[DNSQuery, Endpoint](actionFunc[DNSQuery, Endpoint](getaddrinfoAction))
}

// getaddrinfoAction is the [Action] that resolves domain names using getaddrinfo(3).
func getaddrinfoAction(ctx context.Context, query DNSQuery, outputs chan<- Result[Endpoint]) {
	// start the operation logger
	ol := logx.NewOperationLogger(query.Logger, "Getaddrinfo %s", query.Domain)

	// create resolver
	resolver := netxlite.NewStdlibResolver(query.Logger)

	// getaddrinfo
	addrs, err := resolver.LookupHost(ctx, query.Domain)

	// stop the operation logger
	ol.Stop(err)

	// handle the error case
	if err != nil {
		outputs <- NewResultError[Endpoint](err)
		return
	}

	// handle the successful case
	for _, addr := range addrs {
		res := Endpoint{
			IPAddress: addr,
			Logger:    query.Logger,
			Network:   query.EndpointNetwork,
			Port:      query.EndpointPort,
		}
		outputs <- NewResultValue(res)
	}
}
