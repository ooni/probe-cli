package webconnectivity

//
// DNSResolvers
//
// This code was generated by `boilerplate' using
// the multi-resolver template.
//

import (
	"context"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/atomicx"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// Resolves the URL's domain using several resolvers.
//
// The zero value of this structure IS NOT valid and you MUST initialize
// all the fields marked as MANDATORY before using this structure.
type DNSResolvers struct {
	// Domain is the MANDATORY domain to resolve.
	Domain string

	// IDGenerator is the MANDATORY atomic int64 to generate task IDs.
	IDGenerator *atomicx.Int64

	// Logger is the MANDATORY logger to use.
	Logger model.Logger

	// TestKeys is MANDATORY and contains the TestKeys.
	TestKeys *TestKeys

	// URL is the MANDATORY URL we're measuring.
	URL *url.URL

	// ZeroTime is the MANDATORY zero time of the measurement.
	ZeroTime time.Time

	// WaitGroup is the MANDATORY wait group this task belongs to.
	WaitGroup *sync.WaitGroup

	// DNSOverHTTPSURL is the optional DoH URL to use. If this field is not
	// set, we use a default one (e.g., `https://mozilla.cloudflare-dns.com/dns-query`).
	DNSOverHTTPSURL string

	// UDPAddress is the OPTIONAL address of the UDP resolver to use. If this
	// field is not set we use a default one (e.g., `8.8.8.8:53`).
	UDPAddress string
}

// Start starts this task in a background goroutine.
func (t *DNSResolvers) Start(ctx context.Context) {
	t.WaitGroup.Add(1)
	go func() {
		defer t.WaitGroup.Done() // synchronize with the parent
		t.Run(ctx)
	}()
}

// Run runs this task in the current goroutine.
func (t *DNSResolvers) Run(parentCtx context.Context) {
	// create output channels for the lookup
	systemOut := make(chan []string)
	udpOut := make(chan []string)
	httpsOut := make(chan []string)

	// start asynchronous lookups
	go t.lookupHostSystem(parentCtx, systemOut)
	go t.lookupHostUDP(parentCtx, udpOut)
	go t.lookupHostDNSOverHTTPS(parentCtx, httpsOut)

	// collect resulting IP addresses (which may be nil/empty lists)
	systemAddrs := <-systemOut
	udpAddrs := <-udpOut
	httpsAddrs := <-httpsOut

	// merge the resolved IP addresses
	merged := map[string]bool{}
	for _, addr := range systemAddrs {
		merged[addr] = true
	}
	for _, addr := range udpAddrs {
		merged[addr] = true
	}
	for _, addr := range httpsAddrs {
		merged[addr] = true
	}

	// rearrange addresses to have IPv4 first
	sorted := []string{}
	for addr := range merged {
		if v6, err := netxlite.IsIPv6(addr); err == nil && !v6 {
			sorted = append(sorted, addr)
		}
	}
	for addr := range merged {
		if v6, err := netxlite.IsIPv6(addr); err == nil && v6 {
			sorted = append(sorted, addr)
		}
	}

	// TODO(bassosimone): remove bogons

	// fan out a number of child async tasks to use the IP addrs
	t.startCleartextFlows(parentCtx, sorted)
	t.startSecureFlows(parentCtx, sorted)
}

// lookupHostSystem performs a DNS lookup using the system resolver.
func (t *DNSResolvers) lookupHostSystem(parentCtx context.Context, out chan<- []string) {
	// create context with attached a timeout
	const timeout = 4 * time.Second
	lookupCtx, lookpCancel := context.WithTimeout(parentCtx, timeout)
	defer lookpCancel()

	// create trace's index
	index := t.IDGenerator.Add(1)

	// create trace
	trace := measurexlite.NewTrace(index, t.ZeroTime)

	// start the operation logger
	ol := measurexlite.NewOperationLogger(t.Logger, "DNSResolvers+System#%d: %s", index, t.Domain)

	// runs the lookup
	reso := trace.NewStdlibResolver(t.Logger)
	addrs, err := reso.LookupHost(lookupCtx, t.Domain)
	t.TestKeys.AppendQueries(trace.DNSLookupsFromRoundTrip()...)
	ol.Stop(err)
	out <- addrs
}

// lookupHostUDP performs a DNS lookup using an UDP resolver.
func (t *DNSResolvers) lookupHostUDP(parentCtx context.Context, out chan<- []string) {
	// create context with attached a timeout
	const timeout = 4 * time.Second
	lookupCtx, lookpCancel := context.WithTimeout(parentCtx, timeout)
	defer lookpCancel()

	// create trace's index
	index := t.IDGenerator.Add(1)

	// create trace
	trace := measurexlite.NewTrace(index, t.ZeroTime)

	// start the operation logger
	ol := measurexlite.NewOperationLogger(t.Logger, "DNSResolvers+UDP#%d: %s", index, t.Domain)

	// runs the lookup
	dialer := netxlite.NewDialerWithoutResolver(t.Logger)
	reso := trace.NewParallelUDPResolver(t.Logger, dialer, t.udpAddress())
	addrs, err := reso.LookupHost(lookupCtx, t.Domain)
	t.TestKeys.AppendQueries(trace.DNSLookupsFromRoundTrip()...)
	ol.Stop(err)
	out <- addrs
}

// Returns the UDP resolver we should be using by default.
func (t *DNSResolvers) udpAddress() string {
	if t.UDPAddress != "" {
		return t.UDPAddress
	}
	return "8.8.4.4:53"
}

// lookupHostDNSOverHTTPS performs a DNS lookup using a DoH resolver.
func (t *DNSResolvers) lookupHostDNSOverHTTPS(parentCtx context.Context, out chan<- []string) {
	// create context with attached a timeout
	const timeout = 4 * time.Second
	lookupCtx, lookpCancel := context.WithTimeout(parentCtx, timeout)
	defer lookpCancel()

	// create trace's index
	index := t.IDGenerator.Add(1)

	// create trace
	trace := measurexlite.NewTrace(index, t.ZeroTime)

	// start the operation logger
	ol := measurexlite.NewOperationLogger(t.Logger, "DNSResolvers+DNSOverHTTPS#%d: %s", index, t.Domain)

	// runs the lookup
	reso := trace.NewParallelDNSOverHTTPSResolver(t.Logger, t.dnsOverHTTPSURL())
	addrs, err := reso.LookupHost(lookupCtx, t.Domain)
	reso.CloseIdleConnections()
	t.TestKeys.AppendQueries(trace.DNSLookupsFromRoundTrip()...)
	ol.Stop(err)
	out <- addrs
}

// Returns the DOH resolver URL we should be using by default.
func (t *DNSResolvers) dnsOverHTTPSURL() string {
	if t.DNSOverHTTPSURL != "" {
		return t.DNSOverHTTPSURL
	}
	return "https://mozilla.cloudflare-dns.com/dns-query"
}

// startCleartextFlows starts a TCP measurement flow for each IP addr.
func (t *DNSResolvers) startCleartextFlows(ctx context.Context, addresses []string) {
	if t.URL.Scheme != "http" {
		// Do not bother with measuring HTTP when the user
		// has asked us to measure an HTTPS URL.
		return
	}
	sema := make(chan any, 1)
	sema <- true // allow a single flow to fetch the HTTP body
	port := "80"
	if urlPort := t.URL.Port(); urlPort != "" {
		port = urlPort
	}
	for _, addr := range addresses {
		task := &CleartextFlow{
			Address:     net.JoinHostPort(addr, port),
			IDGenerator: t.IDGenerator,
			Logger:      t.Logger,
			Sema:        sema,
			TestKeys:    t.TestKeys,
			ZeroTime:    t.ZeroTime,
			WaitGroup:   t.WaitGroup,
			HostHeader:  t.URL.Host,
			URLPath:     t.URL.Path,
			URLRawQuery: t.URL.RawQuery,
		}
		task.Start(ctx)
	}
}

// startSecureFlows starts a TCP+TLS measurement flow for each IP addr.
func (t *DNSResolvers) startSecureFlows(ctx context.Context, addresses []string) {
	sema := make(chan any, 1)
	if t.URL.Scheme == "https" {
		// Allows just a single worker to fetch the response body but do that
		// only if the test-lists URL uses "https" as the scheme. Otherwise, just
		// validate IPs by performing a TLS handshake.
		sema <- true
	}
	port := "443"
	if urlPort := t.URL.Port(); urlPort != "" {
		if t.URL.Scheme != "https" {
			// If the URL is like http://example.com:8080/, we don't know
			// which would be the correct port where to use HTTPS.
			return
		}
		port = urlPort
	}
	for _, addr := range addresses {
		task := &SecureFlow{
			Address:     net.JoinHostPort(addr, port),
			IDGenerator: t.IDGenerator,
			Logger:      t.Logger,
			Sema:        sema,
			TestKeys:    t.TestKeys,
			ZeroTime:    t.ZeroTime,
			WaitGroup:   t.WaitGroup,
			ALPN:        []string{"h2", "http/1.1"},
			SNI:         t.URL.Hostname(),
			HostHeader:  t.URL.Host,
			URLPath:     t.URL.Path,
			URLRawQuery: t.URL.RawQuery,
		}
		task.Start(ctx)
	}
}
