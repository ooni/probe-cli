package dnsreport

import (
	"context"
	"net"
	"net/url"
	"sync"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/cmd/gardener/internal/testlists"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// Measurement is the measurement produced by performing a DNS lookup
// of the domain inside a [testlists.Entry].
type Measurement struct {
	// entry is the entry we measured.
	Entry *testlists.Entry

	// addresses contains the resolver addresses.
	Addresses []string

	// failure is the failure that occurred or an empty string.
	Failure *string
}

// measurerWorker is a worker that performs DNS measurements.
func measurerWorker(
	ctx context.Context,
	wg *sync.WaitGroup,
	idx int,
	dnsOverHTTPSServerURL string,
	inputs <-chan *testlists.Entry,
	outputs chan<- *Measurement,
) {
	// logging
	log.Debugf("worker #%d... running", idx)
	defer log.Debugf("worker #%d... done", idx)

	// synchronize with the parent goroutine
	defer wg.Done()

	// create DNS resolver
	dnsTransport := netxlite.NewDNSOverHTTPSTransportWithHTTPTransport(
		defaultHTTPTransport,
		dnsOverHTTPSServerURL,
	)
	reso := netxlite.WrapResolver(
		log.Log,
		netxlite.NewUnwrappedParallelResolver(dnsTransport),
	)

	// walk through the incoming entries
	for entry := range inputs {
		if !entryMeasurer(ctx, entry, reso, outputs) {
			return
		}
	}
}

// entryMeasurer measures a single [testlists.Entry].
func entryMeasurer(
	ctx context.Context,
	entry *testlists.Entry,
	reso model.Resolver,
	outputs chan<- *Measurement,
) bool {
	// parse the URL and skip entries containing IP addresses
	URL := runtimex.Try1(url.Parse(entry.URL))
	if net.ParseIP(URL.Hostname()) != nil {
		return true
	}

	// perform the DNS lookup
	addrs, err := reso.LookupHost(ctx, URL.Hostname())

	// create the related measurement
	measurement := &Measurement{
		Entry:     entry,
		Addresses: addrs,
		Failure:   measurexlite.NewFailure(err),
	}

	// emit the measurement, if possible
	select {
	case <-ctx.Done():
		return false
	case outputs <- measurement:
		return true
	}
}
