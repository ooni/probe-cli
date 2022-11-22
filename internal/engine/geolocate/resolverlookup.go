package geolocate

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

type resolverLookupClient struct {
	Logger model.Logger
}

func (rlc resolverLookupClient) LookupResolverIP(ctx context.Context) (string, error) {
	// MUST be the system resolver! See https://github.com/ooni/probe/issues/2360
	reso := netxlite.NewStdlibResolver(rlc.Logger)
	var ips []string
	ips, err := reso.LookupHost(ctx, "whoami.v4.powerdns.org")
	if err != nil {
		return "", err
	}
	runtimex.Assert(len(ips) >= 1, "reso.LookupHost returned zero IP addresses")
	return ips[0], nil
}
