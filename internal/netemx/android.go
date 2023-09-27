package netemx

import (
	"context"

	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// androidStack wraps [netem.UnderlyingNetwork] to simulate what our getaddrinfo
// wrapper does on Android when it sees the EAI_NODATA return value.
type androidStack struct {
	netem.UnderlyingNetwork
}

// GetaddrinfoLookupANY implements [netem.UnderlyingNetwork]
func (as *androidStack) GetaddrinfoLookupANY(ctx context.Context, domain string) ([]string, string, error) {
	addrs, cname, err := as.UnderlyingNetwork.GetaddrinfoLookupANY(ctx, domain)
	if err != nil {
		err = netxlite.NewErrGetaddrinfo(0, netxlite.ErrAndroidDNSCacheNoData)
	}
	return addrs, cname, err
}
