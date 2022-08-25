package tlsmiddlebox

//
// DNS Lookup for tlsmiddlebox
//

import (
	"context"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// DNSLookup performs a DNS Lookup for the passed domain
func (m *Measurer) DNSLookup(ctx context.Context, index int64, zeroTime time.Time,
	logger model.Logger, domain string, tk *TestKeys) ([]string, error) {
	url := m.config.resolverURL()
	trace := measurexlite.NewTrace(index, zeroTime)
	ol := measurexlite.NewOperationLogger(logger, "DNSLookup #%d, %s, %s", index, url, domain)
	// TODO(DecFox): We are currently using the DoH resolver, we will
	// switch to the TRR2 resolver once we have it in measurexlite
	// Issue: https://github.com/ooni/probe/issues/2185
	resolver := trace.NewParallelDNSOverHTTPSResolver(logger, url)
	addrs, err := resolver.LookupHost(ctx, domain)
	ol.Stop(err)
	tk.addQueries(trace.DNSLookupsFromRoundTrip())
	return addrs, err
}
